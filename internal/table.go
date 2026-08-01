package internal

type TableDefinition struct {
	name    string
	columns []Column
}

type Column struct {
	name     string
	dataType DataType
}

type DataType uint8

const (
	NullType DataType = iota + 1
	IntegerType
	RealType
	TextType
	BoolType
	BlobType
	TimeType
	UUIDType
)

func (t *TableDefinition) Encode() []byte {
	e := NewEncoder()

	e.PutShortStringWithLength(t.name)

	e.PutInt32(int32(len(t.columns)))

	for _, col := range t.columns {
		e.PutShortStringWithLength(col.name)
		e.PutDataType(col.dataType)
	}

	return e.Bytes()
}

func DecodeTableDefinition(buf []byte) (*TableDefinition, error) {
	d := NewDecoder(buf)

	name, err := d.GetShortStringWithLength()

	if err != nil {
		return nil, err
	}

	numCols, err := d.GetInt32()

	if err != nil {
		return nil, err
	}

	td := &TableDefinition{
		name:    name,
		columns: []Column{},
	}

	for range numCols {
		colName, err := d.GetShortStringWithLength()

		if err != nil {
			return nil, err
		}

		dt, err := d.GetDataType()

		if err != nil {
			return nil, err
		}

		td.columns = append(td.columns, Column{name: colName, dataType: dt})
	}

	return td, nil
}

func (td *TableDefinition) ColumnNames() []string {
	names := []string{}

	for _, col := range td.columns {
		names = append(names, col.name)
	}

	return names
}
