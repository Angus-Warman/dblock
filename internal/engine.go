package internal

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/Angus-Warman/dblock/internal/metadata"
	"github.com/Angus-Warman/dblock/internal/pragma"
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
		return -1, -1, fmt.Errorf("nothing to execute")
	}

	if stmt.createStmt != nil {
		err := e.ExecCreate(stmt.createStmt)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil // TODO technically a row was created in dblock_schema, with an ID
	}

	if stmt.insertStmt != nil {
		return e.Insert(stmt.insertStmt, args)
	}

	if stmt.updateStmt != nil {
		rowsAffected, err := e.Update(stmt.updateStmt, args)

		if err != nil {
			return -1, -1, err
		}

		return -1, rowsAffected, nil
	}

	if stmt.alterStmt != nil {
		err := e.AlterTable(stmt.alterStmt)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil
	}

	if stmt.dropStmt != nil {
		err := e.DropTable(stmt.dropStmt.tableName, stmt.dropStmt.ifExists)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil
	}

	if stmt.pragmaStmt != nil {
		err := e.execPragma(stmt.pragmaStmt)

		if err != nil {
			return -1, -1, err
		}

		return 0, 0, nil
	}

	if stmt.createIdxStmt != nil {
		err := e.createIndex(stmt.createIdxStmt)

		if err != nil {
			return 0, 0, err
		}

		return 0, 0, nil
	}

	return -1, -1, fmt.Errorf("unsupported statement")
}

func (e *Engine) Query(stmt *QueryStmt, args []any) (Scanner, error) {
	if stmt == nil {
		return nil, fmt.Errorf("nothing to query in stmt")
	}

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
		v, err := strconv.ParseInt(stmt.value, 10, 64)

		if err != nil {
			return fmt.Errorf("execPragma page_size: %w", err)
		}

		if m.NumberOfPages != 0 {
			return fmt.Errorf("page_size can only be set before first page write")
		}

		min := int64(8)  // 256 bytes
		max := int64(22) // 4 MB

		if v < min {
			return fmt.Errorf("min page size %v", min)
		}

		if v > max {
			return fmt.Errorf("max page size %v", max)
		}

		m.PageSizePower = uint8(v)

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

	if stmt.property == pragma.ReadMetadata {
		rows := [][]any{
			{"dblock", m.Dblock},
			{"file_version", int64(m.FileVersion)},
			{"schema_version", int64(m.SchemaVersion)},
			{"page_size_power", int64(m.PageSizePower)},
			{"number_of_pages", int64(m.NumberOfPages)},
			{"token", m.Token},
			{"checksum", int64(m.Checksum)},
		}

		if len(rows) != metadata.NumProperties {
			return nil, fmt.Errorf("should have %v properties", metadata.NumProperties)
		}

		return NewCustomScanner([]string{"property", "value"}, rows), nil
	}

	var val any

	switch stmt.property {
	case pragma.TokenProperty:
		val = m.Token

	case pragma.PageSizeProperty:
		val = int64(m.PageSize())

	default:
		return nil, fmt.Errorf("queryPragma: unsupported pragma %q", stmt.property)
	}

	return NewCustomScanner([]string{string(stmt.property)}, [][]any{{val}}), nil
}

// hasAggregates reports whether the statement needs aggregation: a GROUP BY
// clause, or any aggregate function call in the projection.
func hasAggregates(stmt *SelectStmt) bool {
	if len(stmt.groupBy) > 0 {
		return true
	}

	for _, item := range stmt.projection {
		if item.expr != nil && item.expr.Kind == FuncKind {
			return true
		}
	}

	return false
}

func (e *Engine) CreateFullScanner(tableName string) (Scanner, error) {
	if tableName == "dblock_schema" {
		rootpage := 0
		tree := NewBtree(e.pager, PageID(rootpage))
		return NewFullScanner(tree, schemaColumns)
	}

	info, err := e.lookupTable(tableName)

	if err != nil {
		return nil, fmt.Errorf("create full scanner: %w", err)
	}

	tree := NewBtree(e.pager, info.rootPage)

	return NewFullScanner(tree, info.columns)
}

type tableInfo struct {
	rootPage PageID
	columns  []string
	table    *Table
}

func (e *Engine) lookupTable(name string) (*tableInfo, error) {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	_, encodedRows, err := rootTree.All()

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

	return nil, fmt.Errorf("table '%v' does not exist", name)
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

	rowValues, err := buildRowValues(info.table.columns, stmt.columns, values)

	if err != nil {
		return -1, -1, err
	}

	for i, col := range info.table.columns {
		coerced, err := coerceValue(col, rowValues[i])

		if err != nil {
			return -1, -1, fmt.Errorf("Insert: %w", err)
		}

		rowValues[i] = coerced
	}

	row := Row{Values: rowValues}
	encodedRow := row.Encode()
	tree := NewBtree(e.pager, info.rootPage)
	rowID, err := tree.InsertNext(encodedRow)

	if err != nil {
		return -1, -1, err
	}

	rowsAffected = 1

	idxs, err := e.findIndexes(stmt.tableName)

	if err != nil {
		return 0, 0, err
	}

	for _, idx := range idxs {
		err := idx.Insert(rowID, row)

		if err != nil {
			return 0, 0, err
		}

		rowsAffected += 1
	}

	return int64(rowID), rowsAffected, nil
}

// buildRowValues maps supplied insert values onto full row values. With named
// columns each value targets that column; otherwise values are positional.
// Missing columns and DEFAULT keywords are filled from the column definition,
// evaluated left-to-right so a default may reference earlier columns.
func buildRowValues(columns []Column, named []string, values []any) ([]any, error) {
	if len(named) == 0 {
		if len(values) > len(columns) {
			return nil, fmt.Errorf("Insert: expected %d values, got %d", len(columns), len(values))
		}
	} else if len(values) > len(named) {
		return nil, fmt.Errorf("Insert: expected %d values, got %d", len(named), len(values))
	}

	colNames := make([]string, len(columns))

	for i, col := range columns {
		colNames[i] = col.name
	}

	pending := make(map[int]any, len(values))

	if len(named) == 0 {
		for i, v := range values {
			pending[i] = v
		}
	} else {
		colIdx := make(map[string]int, len(columns))

		for i, col := range columns {
			colIdx[col.name] = i
		}

		for i, v := range values {
			idx, ok := colIdx[named[i]]

			if !ok {
				return nil, fmt.Errorf("Insert: no such column %q", named[i])
			}

			pending[idx] = v
		}
	}

	row := make([]any, len(columns))

	for i := range columns {
		v, supplied := pending[i]

		if supplied {
			if _, isDefault := v.(DefaultKeyword); isDefault {
				dv, err := evalDefaultExpr(columns[i], colNames, row)

				if err != nil {
					return nil, err
				}

				row[i] = dv
				continue
			}

			row[i] = v
			continue
		}

		dv, err := evalDefaultExpr(columns[i], colNames, row)

		if err != nil {
			return nil, err
		}

		row[i] = dv
	}

	return row, nil
}

func (e *Engine) findIndexes(tableName string) ([]*Index, error) {
	objs, err := e.GetSchemaObjects(IndexObject)

	if err != nil {
		return nil, err
	}

	out := []*Index{}

	for _, obj := range objs {
		def, err := DecodeIndexDefinition(obj.definition)

		if err != nil {
			return nil, err
		}

		if def.TableName != tableName {
			continue
		}

		table, err := e.lookupTable(def.TableName)

		if err != nil {
			return nil, err
		}

		tree := NewBtree(e.pager, obj.rootpage)

		idx, err := IndexFromDefinition(*def, table.table, tree)

		if err != nil {
			return nil, err
		}

		out = append(out, idx)
	}

	return out, nil
}

// Update applies an expression to the target column of every row matching
// the WHERE clause, returning the number of rows modified.
func (e *Engine) Update(stmt *UpdateStmt, args []any) (int64, error) {
	if stmt == nil {
		return -1, fmt.Errorf("Update: nothing to update in statement")
	}

	info, err := e.lookupTable(stmt.tableName)

	if err != nil {
		return -1, err
	}

	colIdx := -1

	for i, name := range info.columns {
		if name == stmt.column {
			colIdx = i
			break
		}
	}

	if colIdx < 0 {
		return -1, fmt.Errorf("Update: no such column %q", stmt.column)
	}

	idxs, err := e.findIndexes(stmt.tableName)

	if err != nil {
		return -1, err
	}

	affectedIdxs := []*Index{}

	for _, idx := range idxs {
		if idx.containsColumn(stmt.column) {
			affectedIdxs = append(affectedIdxs, idx)
		}
	}

	tree := NewBtree(e.pager, info.rootPage)
	keys, encodedRows, err := tree.All()

	if err != nil {
		return -1, err
	}

	var rowsAffected int64

	for i, encodedRow := range encodedRows {
		originalRow, err := DecodeRow(encodedRow)

		if err != nil {
			return -1, err
		}

		if stmt.where != nil {
			val, err := evalExpr(stmt.where, info.columns, originalRow.Values, args)

			if err != nil {
				return -1, err
			}

			matches, isBool := val.(bool)

			if !isBool {
				return -1, fmt.Errorf("Update: where: expected boolean expression, got %T", val)
			}

			if !matches {
				continue
			}
		}

		updatedRow := originalRow.Clone()

		val, err := evalExpr(stmt.expr, info.columns, originalRow.Values, args)

		if err != nil {
			return -1, err
		}

		coerced, err := coerceValue(info.table.columns[colIdx], val)

		if err != nil {
			return -1, fmt.Errorf("Update: %w", err)
		}

		updatedRow.Values[colIdx] = coerced

		if err := tree.Insert(keys[i], updatedRow.Encode()); err != nil {
			return -1, err
		}

		for _, idx := range affectedIdxs {
			err = idx.UpdateIndexEntry(originalRow, updatedRow)

			if err != nil {
				return -1, err
			}
		}

		rowsAffected++
	}

	return rowsAffected, nil
}

// AlterTable applies an ALTER TABLE operation to a stored table definition.
func (e *Engine) AlterTable(stmt *AlterStmt) error {
	if stmt == nil {
		return fmt.Errorf("AlterTable: nothing to alter in statement")
	}

	info, err := e.lookupTable(stmt.tableName)

	if err != nil {
		if stmt.ifExists {
			return nil
		}

		return err
	}

	table := info.table

	switch {
	case stmt.alterCol != nil:
		if err := e.alterColumnType(table, stmt.alterCol); err != nil {
			return err
		}

	case stmt.renameCol != nil:
		if err := e.renameColumn(table, stmt.renameCol); err != nil {
			return err
		}

	case stmt.renameTbl != nil:
		if err := e.checkRenameTable(stmt.tableName, stmt.renameTbl.newName); err != nil {
			return err
		}

		e.renameTable(table, stmt.renameTbl)

	case stmt.addCol != nil:
		if err := e.addColumn(table, stmt.addCol, info.rootPage); err != nil {
			return err
		}

	default:
		return fmt.Errorf("AlterTable: unsupported ALTER TABLE operation")
	}

	return e.updateSchemaTable(stmt.tableName, table, info.rootPage)
}

func (e *Engine) alterColumnType(table *Table, op *AlterColumnTypeOp) error {
	colIdx := -1

	for i, col := range table.columns {
		if col.name == op.colName {
			colIdx = i
			break
		}
	}

	if colIdx < 0 {
		return fmt.Errorf("ALTER TABLE: no such column %q", op.colName)
	}

	dt, err := parseDataType(op.newType)

	if err != nil {
		return err
	}

	// TODO: actually check if this would cause data loss
	if table.columns[colIdx].dataType == AnyType && dt != AnyType {
		return fmt.Errorf("ALTER COLUMN %q: changing type from ANY to %s could cause data loss", op.colName, dt)
	}

	table.columns[colIdx].dataType = dt

	return nil
}

// addColumn appends a column to a table definition and fills existing rows
// with the column's default (or the type's zero value when it has none).
func (e *Engine) addColumn(table *Table, op *AddColumnOp, rootPage PageID) error {
	for _, col := range table.columns {
		if col.name == op.column.name {
			return fmt.Errorf("ALTER TABLE: column %q already exists", op.column.name)
		}
	}

	table.columns = append(table.columns, op.column)

	value, err := evalDefaultExpr(op.column, table.ColumnNames(), nil)

	if err != nil {
		return err
	}

	if value == nil {
		value = zeroValue(op.column.dataType)
	}

	coerced, err := coerceValue(op.column, value)

	if err != nil {
		return fmt.Errorf("ALTER TABLE: %w", err)
	}

	tree := NewBtree(e.pager, rootPage)
	keys, encodedRows, err := tree.All()

	if err != nil {
		return err
	}

	for i, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return err
		}

		row.Values = append(row.Values, coerced)

		if err := tree.Insert(keys[i], row.Encode()); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) renameColumn(table *Table, op *RenameColumnOp) error {
	for i, col := range table.columns {
		if col.name == op.oldName {
			table.columns[i].name = op.newName
			return nil
		}
	}

	return fmt.Errorf("ALTER TABLE: no such column %q", op.oldName)
}

func (e *Engine) renameTable(table *Table, op *RenameTableOp) error {
	table.name = op.newName
	return nil
}

func (e *Engine) checkRenameTable(oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	rs, err := e.GetRootSchema()

	if err != nil {
		return err
	}

	if slices.Contains(rs.tableNames, newName) {
		return fmt.Errorf("table '%v' already exists", newName)
	}

	return nil
}

func (e *Engine) ExecCreate(stmt *CreateStmt) error {
	def := &Table{
		name:    stmt.tableName,
		columns: stmt.columns,
	}

	if err := e.CreateTable(def); err != nil {
		return err
	}

	for _, col := range stmt.uniqueCols {
		if err := e.createIndexObject(def, fmt.Sprintf("index_%s_%s", def.name, col), []string{col}, true); err != nil {
			return err
		}
	}

	return nil
}

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

	obj := &SchemaObject{
		name:       def.name,
		objectType: TableObject,
		definition: def.Encode(),
		rootpage:   rootpage,
	}

	return e.SaveSchemaObject(obj)
}

func (e *Engine) DropTable(name string, ifExists bool) error {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	keys, encodedRows, err := rootTree.All()

	if err != nil {
		return err
	}

	keyIdx := -1

	for i, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return err
		}

		if row.Values[0].(string) == name {
			keyIdx = i
			break
		}
	}

	if keyIdx < 0 {
		if ifExists {
			// Not found
			return nil
		}

		return fmt.Errorf("'%s' does not exist", name)
	}

	err = rootTree.Delete(keys[keyIdx])

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

func (e *Engine) createIndex(stmt *CreateIdxStmt) error {
	table, err := e.lookupTable(stmt.tableName)

	if err != nil {
		return err
	}

	colLookup := make(map[string]bool)

	for _, col := range table.columns {
		colLookup[col] = true
	}

	for _, col := range stmt.columnNames {
		exists := colLookup[col]

		if !exists {
			return fmt.Errorf("column %v does not exist on table %v", col, stmt.tableName)
		}
	}

	return e.createIndexObject(table.table, stmt.idxName, stmt.columnNames, stmt.unique)
}

func (e *Engine) createIndexObject(table *Table, idxName string, columnNames []string, unique bool) error {
	idxDef := &IndexDefinition{
		TableName: table.name,
		Unique:    unique,
		Columns:   columnNames,
	}

	rootpage := e.pager.NextID()

	obj := &SchemaObject{
		name:       idxName,
		objectType: IndexObject,
		definition: idxDef.Encode(),
		rootpage:   rootpage,
	}

	err := e.SaveSchemaObject(obj)

	if err != nil {
		return err
	}

	idxTree := NewBtree(e.pager, obj.rootpage)

	idx, err := IndexFromDefinition(*idxDef, table, idxTree)

	if err != nil {
		return err
	}

	scanner, err := e.CreateFullScanner(table.name)

	if err != nil {
		return err
	}

	// Insert all rows into new index
	for {
		k, r, ok, err := scanner.Next()
		if !ok {
			break
		}
		if err != nil {
			return err
		}

		id := DecodeKey(k)

		if err = idx.Insert(id, r); err != nil {
			return err
		}
	}

	return nil
}
