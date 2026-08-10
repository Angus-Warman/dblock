package internal

import "fmt"

type SchemaObjectType string

const (
	TableObject SchemaObjectType = "TABLE"
	IndexObject SchemaObjectType = "INDEX"
)

type SchemaObject struct {
	objectType SchemaObjectType
	name       string
	definition []byte
	rootpage   PageID
}

const (
	schemaNameColumn = "name"
	objectTypeColumn = "object_type"
	definitionColumn = "definition"
	rootpageColumn   = "rootPage"
)

var schemaColumns = []string{
	schemaNameColumn,
	objectTypeColumn,
	definitionColumn,
	rootpageColumn,
}

type RootSchema struct {
	tableNames []string
}

func (e *Engine) GetRootSchema() (*RootSchema, error) {
	tableObjs, err := e.GetSchemaObjects(TableObject)

	if err != nil {
		return nil, err
	}

	tableNames := []string{}

	for _, obj := range tableObjs {
		tableNames = append(tableNames, obj.name)
	}

	return &RootSchema{
		tableNames: tableNames,
	}, nil
}

func (e *Engine) GetAllSchemaObjects() ([]SchemaObject, error) {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	_, encodedRows, err := rootTree.All()

	if err != nil {
		return nil, err
	}

	objects := []SchemaObject{}

	for _, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return nil, err
		}

		objName, err := GetRowValueAs[string](&row, 0)

		if err != nil {
			return nil, err
		}

		objType, err := GetRowValueAs[string](&row, 1)

		if err != nil {
			return nil, err
		}

		def, err := GetRowValueAs[[]byte](&row, 2)

		if err != nil {
			return nil, err
		}

		rootpage, err := GetRowValueAs[int64](&row, 3)

		if err != nil {
			return nil, err
		}

		objects = append(objects, SchemaObject{
			name:       objName,
			objectType: SchemaObjectType(objType), // TODO should really check this
			definition: def,
			rootpage:   PageID(rootpage),
		})
	}

	return objects, nil
}

func (e *Engine) GetSchemaObjects(objType SchemaObjectType) ([]SchemaObject, error) {
	all, err := e.GetAllSchemaObjects()

	if err != nil {
		return nil, err
	}

	out := []SchemaObject{}

	for _, obj := range all {
		if obj.objectType == objType {
			out = append(out, obj)
		}
	}

	return out, nil
}

func (e *Engine) SaveSchemaObject(obj *SchemaObject) error {
	values := []any{
		obj.name, string(obj.objectType), obj.definition, int64(obj.rootpage),
	}

	row := Row{Values: values}
	encodedRow := row.Encode()
	rootTree := NewBtree(e.pager, RootSchemaPageID)
	_, err := rootTree.InsertNext(encodedRow)

	if err != nil {
		return err
	}

	return e.bumpSchemaVersion() // TODO: This should probably be moved out, since multiple objects can be saved in one go
}

// updateSchemaTable replaces the schema entry for schemaName with the
// current table definition, preserving its row ID in dblock_schema.
func (e *Engine) updateSchemaTable(schemaName string, table *Table, rootPage PageID) error {
	rootTree := NewBtree(e.pager, RootSchemaPageID)

	keys, encodedRows, err := rootTree.All()

	if err != nil {
		return err
	}

	for i, encodedRow := range encodedRows {
		row, err := DecodeRow(encodedRow)

		if err != nil {
			return err
		}

		if row.Values[0].(string) != schemaName {
			continue
		}

		row.Values[0] = table.name
		row.Values[1] = string(TableObject)
		row.Values[2] = table.Encode()
		row.Values[3] = int64(rootPage)

		if err := rootTree.Insert(keys[i], row.Encode()); err != nil {
			return err
		}

		return e.bumpSchemaVersion()
	}

	return fmt.Errorf("table '%v' does not exist", schemaName)
}
