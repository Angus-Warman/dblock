package internal

import (
	"dblock2/internal/pragma"
	"fmt"
	"slices"
	"strconv"
)

type Engine struct {
	pager Pager
}

func NewEngine(pager Pager) (*Engine, error) {
	if pager == nil {
		return nil, fmt.Errorf("pager is null")
	}

	return &Engine{
		pager: pager,
	}, nil
}

func (e *Engine) Exec(stmt *ExecStmt, args []any) (insertedID int64, rowsAffected int64, err error) {
	if stmt == nil {
		return -1, -1, fmt.Errorf("Exec: nothing to execute in statement")
	}

	if stmt.createStmt != nil {
		err := e.ExecCreate(stmt.createStmt)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil
	}

	if stmt.insertStmt != nil {
		return e.Insert(stmt.insertStmt, args)
	}

	if stmt.pragmaStmt != nil {
		err := e.execPragma(stmt.pragmaStmt)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil
	}

	return -1, -1, fmt.Errorf("Exec: unsupported statement")
}

type RootSchema struct {
	tableNames []string
}

func (e *Engine) GetRootSchema() (*RootSchema, error) {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	encodedRows, err := rootTree.All()

	if err != nil {
		return nil, err
	}

	tableNames := []string{}

	for _, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return nil, fmt.Errorf("GetRootSchema: %w", err)
		}

		tableName := row.Values[0].(string) // TODO
		tableNames = append(tableNames, tableName)
	}

	return &RootSchema{
		tableNames: tableNames,
	}, nil
}

func (e *Engine) Query(stmt *QueryStmt, args []any) (Scanner, error) {
	if stmt.selectStmt != nil {
		return e.querySelect(stmt.selectStmt, args)
	}

	if stmt.pragmaStmt != nil {
		return e.queryPragma(stmt.pragmaStmt)
	}

	return nil, fmt.Errorf("nothing to query in stmt")
}

func (e *Engine) execPragma(stmt *PragmaStmt) error {
	m, err := e.pager.GetMetadata()

	if err != nil {
		return err
	}

	switch stmt.property {
	case pragma.TokenProperty:
		v, err := strconv.ParseInt(stmt.value, 10, 64)

		if err != nil {
			return fmt.Errorf("execPragma token: %w", err)
		}

		m.Token = v

	case pragma.PageSizeProperty:
		v, err := strconv.ParseUint(stmt.value, 10, 32)

		if err != nil {
			return fmt.Errorf("execPragma page_size: %w", err)
		}

		if v != PageSize {
			return fmt.Errorf("execPragma page_size: unsupported page size %d", v)
		}

	default:
		return fmt.Errorf("execPragma: unsupported pragma %q", stmt.property)
	}

	m.CalculateChecksum()

	return e.pager.PutMetadata(m)
}

func (e *Engine) queryPragma(stmt *PragmaStmt) (Scanner, error) {
	m, err := e.pager.GetMetadata()

	if err != nil {
		return nil, err
	}

	var val any

	switch stmt.property {
	case pragma.TokenProperty:
		val = m.Token

	case pragma.PageSizeProperty:
		val = int64(PageSize)

	default:
		return nil, fmt.Errorf("queryPragma: unsupported pragma %q", stmt.property)
	}

	return NewPragmaScanner(stmt.property, val), nil
}

func (e *Engine) querySelect(stmt *SelectStmt, args []any) (Scanner, error) {
	scanner, err := e.SelectAllFromTable(stmt.tableName)

	if err != nil {
		return nil, err
	}

	for _, join := range stmt.joins {
		right, err := e.SelectAllFromTable(join.tableName)

		if err != nil {
			return nil, err
		}

		scanner, err = NewJoinScanner(scanner, right, stmt, &join)

		if err != nil {
			return nil, err
		}
	}

	if len(stmt.orders) > 0 {
		scanner, err = NewOrderScanner(scanner, stmt)

		if err != nil {
			return nil, err
		}
	}

	scanner, err = NewProjectorScanner(scanner, stmt)

	if err != nil {
		return nil, err
	}

	return scanner, nil
}

func (e *Engine) SelectAllFromTable(tableName string) (Scanner, error) {
	if tableName == "dblock_schema" {
		rootpage := 0
		tree := NewBtree(e.pager, PageID(rootpage))
		return NewFullScanner(tree, schemaColumns)
	}

	info, err := e.lookupTable(tableName)

	if err != nil {
		return nil, fmt.Errorf("SelectAllFromTable: %w", err)
	}

	tree := NewBtree(e.pager, info.rootPage)

	return NewFullScanner(tree, info.columns)
}

var schemaColumns = []string{"object_name", "object_type", "definition", "rootpage"}

type tableInfo struct {
	rootPage PageID
	columns  []string
	table    *Table
}

func (e *Engine) lookupTable(name string) (*tableInfo, error) {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	encodedRows, err := rootTree.All()

	if err != nil {
		return nil, err
	}

	for _, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return nil, fmt.Errorf("lookupTable: %w", err)
		}

		if row.Values[0].(string) != name {
			continue
		}

		td, err := DecodeTable(row.Values[2].([]byte))

		if err != nil {
			return nil, err
		}

		colNames := td.ColumnNames()

		return &tableInfo{
			rootPage: PageID(row.Values[3].(int64)),
			columns:  colNames,
			table:    td,
		}, nil
	}

	return nil, fmt.Errorf("no such table: %q", name)
}

func (e *Engine) Insert(stmt *InsertStmt, args []any) (insertedID int64, rowsAffected int64, err error) {
	if stmt == nil {
		return -1, -1, fmt.Errorf("Insert: nothing to insert in statement")
	}

	info, err := e.lookupTable(stmt.tableName)

	if err != nil {
		return -1, -1, err
	}

	values := make([]any, 0, len(stmt.values))
	argIdx := 0

	for _, v := range stmt.values {
		if v == "?" {
			if argIdx >= len(args) {
				return -1, -1, fmt.Errorf("Insert: missing argument for placeholder %d", argIdx+1)
			}

			values = append(values, args[argIdx])
			argIdx++
			continue
		}

		values = append(values, v)
	}

	if len(values) != len(info.table.columns) {
		return -1, -1, fmt.Errorf("Insert: expected %d values, got %d", len(info.table.columns), len(values))
	}

	for i, col := range info.table.columns {
		coerced, err := coerceValue(col, values[i])

		if err != nil {
			return -1, -1, fmt.Errorf("Insert: %w", err)
		}

		values[i] = coerced
	}

	row := Row{Values: values}
	encodedRow := row.Encode()
	tree := NewBtree(e.pager, info.rootPage)
	rowID, err := tree.InsertNext(encodedRow)

	if err != nil {
		return -1, -1, err
	}

	return int64(rowID), 1, nil
}

func (e *Engine) ExecCreate(stmt *CreateStmt) error {
	def := &Table{
		name:    stmt.tableName,
		columns: stmt.columns,
	}

	return e.CreateTable(def)
}

type SchemaObjectType string

const (
	TableObject SchemaObjectType = "TABLE"
)

func (e *Engine) CreateTable(def *Table) error {
	if def == nil {
		return fmt.Errorf("table definition is null")
	}

	rs, err := e.GetRootSchema()

	if err != nil {
		return err
	}

	if slices.Contains(rs.tableNames, def.name) {
		return fmt.Errorf("%q already exists", def.name)
	}

	rootpage := e.pager.NextID()

	return e.SaveSchemaObject(def.name, TableObject, def.Encode(), rootpage)
}

func (e *Engine) SaveSchemaObject(objectName string, objectType SchemaObjectType, definition []byte, rootpage PageID) error {
	values := []any{
		objectName, string(objectType), definition, int64(rootpage),
	}

	row := Row{Values: values}
	encodedRow := row.Encode()
	rootTree := NewBtree(e.pager, RootSchemaPageID)
	_, err := rootTree.InsertNext(encodedRow)

	if err != nil {
		return err
	}

	return e.bumpSchemaVersion()
}

// bumpSchemaVersion records that dblock_schema changed.
func (e *Engine) bumpSchemaVersion() error {
	m, err := e.pager.GetMetadata()

	if err != nil {
		return err
	}

	m.SchemaVersion++

	return e.pager.PutMetadata(m)
}
