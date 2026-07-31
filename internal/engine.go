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

	def := &TableDefinition{
		name: stmt.tableName,
	}

	err = e.CreateTable(def)

	if err != nil {
		return -1, -1, err
	}

	return 0, 0, nil
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

	return e.SaveSchemaObject(def.name, "TABLE", []byte{}, rootpage)
}

func (e *Engine) SaveSchemaObject(objectName, objectType string, definition []byte, rootpage PageID) error {
	values := []any{
		objectName, objectType, definition, int64(rootpage),
	}

	row := Row{Values: values}
	encodedRow := row.Encode()
	key := []byte(objectName)
	rootTree := NewBtree(e.pager, RootSchemaPageID)
	return rootTree.Insert(key, encodedRow)
}
