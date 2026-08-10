package internal

import "github.com/Angus-Warman/dblock/internal/codec"

type Table struct {
	name    string
	columns []Column
}

type Column struct {
	name     string
	dataType DataType
}

func (t *Table) Encode() []byte {
	e := codec.NewEncoder()

	e.PutShortStringWithLength(t.name)

	e.PutAsUint32(len(t.columns))

	for _, col := range t.columns {
		e.PutShortStringWithLength(col.name)
		PutDataType(e, col.dataType)
	}

	return e.Bytes()
}

func DecodeTable(buf []byte) (*Table, error) {
	d := codec.NewDecoder(buf)

	name, err := d.GetShortStringWithLength()

	if err != nil {
		return nil, err
	}

	numCols, err := d.GetInt32()

	if err != nil {
		return nil, err
	}

	td := &Table{
		name:    name,
		columns: []Column{},
	}

	for range numCols {
		colName, err := d.GetShortStringWithLength()

		if err != nil {
			return nil, err
		}

		dt, err := GetDataType(d)

		if err != nil {
			return nil, err
		}

		td.columns = append(td.columns, Column{name: colName, dataType: dt})
	}

	return td, nil
}

func (td *Table) ColumnNames() []string {
	names := []string{}

	for _, col := range td.columns {
		names = append(names, col.name)
	}

	return names
}
