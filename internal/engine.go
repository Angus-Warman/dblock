package internal

import (
	"fmt"
	"slices"
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
	return e.SelectAllFromTable(stmt.tableName)
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

		td, err := DecodeTableDefinition(row.Values[2].([]byte))

		if err != nil {
			return nil, err
		}

		colNames := td.ColumnNames()

		return &tableInfo{
			rootPage: PageID(row.Values[3].(int64)),
			columns:  colNames,
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
	columns := make([]Column, len(stmt.columnNames))
	for i, name := range stmt.columnNames {
		columns[i] = Column{name: name}
	}

	def := &TableDefinition{
		name:    stmt.tableName,
		columns: columns,
	}

	return e.CreateTable(def)
}

type SchemaObjectType string

const (
	TableObject SchemaObjectType = "TABLE"
)

func (e *Engine) CreateTable(def *TableDefinition) error {
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
	return err
}
