package internal

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Angus-Warman/dblock/internal/codec"
)

type IndexDefinition struct {
	TableName string
	Unique    bool
	Columns   []string
}

func (i *IndexDefinition) Encode() []byte {
	e := codec.NewEncoder()

	e.PutShortStringWithLength(i.TableName)

	e.PutBool(i.Unique)

	e.PutAsUint8(len(i.Columns))

	for _, col := range i.Columns {
		e.PutShortStringWithLength(col)
	}

	return e.Bytes()
}

func DecodeIndexDefinition(buf []byte) (*IndexDefinition, error) {
	d := codec.NewDecoder("index definition", buf)

	i := &IndexDefinition{}

	var err error

	i.TableName, err = d.GetShortStringWithLength()

	if err != nil {
		return nil, err
	}

	i.Unique, err = d.GetBool()

	if err != nil {
		return nil, err
	}

	numCols, err := d.GetUint8()

	if err != nil {
		return nil, err
	}

	cols := []string{}

	for range numCols {
		col, err := d.GetShortStringWithLength()

		if err != nil {
			return nil, err
		}

		cols = append(cols, col)
	}

	i.Columns = cols

	return i, nil
}

type Index struct {
	unique    bool
	colMap    map[string]int
	onColumns []string
	idxTree   *Tree
}

func IndexFromDefinition(def IndexDefinition, table *Table, idxTree *Tree) (*Index, error) {
	idx := &Index{
		unique:    def.Unique,
		colMap:    make(map[string]int),
		onColumns: def.Columns,
		idxTree:   idxTree,
	}

	for i, col := range table.columns {
		idx.colMap[col.name] = i
	}

	return idx, nil
}

func (i *Index) ComputeKey(row Row) ([]byte, error) {
	keyValues := make([]any, len(i.onColumns))

	for j, col := range i.onColumns {
		value, err := row.GetValue(i.colMap, col)

		if err != nil {
			return nil, err
		}

		keyValues[j] = value
	}

	computedKey := EncodeMixedArray(keyValues)

	return computedKey, nil
}

func EncodeMixedArray(values []any) []byte {
	e := codec.NewEncoder()

	e.PutAsUint8(len(values))

	for _, value := range values {
		switch v := value.(type) {
		case int64:
			e.PutInt64(v)
		case float64:
			e.PutFloat64(v)
		case time.Time:
			e.PutTime(v)
		case string:
			e.PutStringWithLength(v)
		case []byte:
			e.PutBytesWithLength(v)
		case nil:
			PutDataType(e, NullType)
		}
	}

	return e.Bytes()
}

func (i *Index) Insert(id RowID, row Row) error {
	computed, err := i.ComputeKey(row)

	if err != nil {
		return err
	}

	err = i.checkUnique(computed)

	if err != nil {
		if err == ErrUniqueKey {
			return fmt.Errorf("could not insert %v, unique key already exists", computed)
		}

		return err
	}

	encodedRowID := EncodeKey(id)

	err = i.idxTree.Insert(computed, encodedRowID)

	if err != nil {
		return err
	}

	return nil
}

var ErrUniqueKey = errors.New("unique key already exists")

func (i *Index) checkUnique(key []byte) error {
	if !i.unique {
		return nil
	}

	exists, err := i.idxTree.Contains(key)

	if err != nil {
		return err
	}

	if exists {
		return ErrUniqueKey
	}

	return nil
}

func (i *Index) Lookup(row Row) (RowID, bool, error) {
	computedKey, err := i.ComputeKey(row)

	if err != nil {
		return 0, false, err
	}

	encRowID, found, err := i.idxTree.Search(computedKey)

	if err != nil {
		return 0, false, err
	}

	if !found {
		return 0, found, nil
	}

	rowID := DecodeKey(encRowID)

	return rowID, found, nil
}

func (i *Index) containsColumn(column string) bool {
	return slices.Contains(i.onColumns, column) // TODO: Could be a map here instead
}

func (i *Index) UpdateIndexEntry(originalRow Row, updatedRow Row) error {
	originalKey, err := i.ComputeKey(originalRow)

	if err != nil {
		return err
	}

	rowID, found, err := i.Lookup(originalRow)

	if err != nil {
		return err
	}

	if !found {
		// TODO
		return fmt.Errorf("tried to update an index entry, but the original entry did not exist")
	}

	err = i.idxTree.Delete(originalKey)

	if err != nil {
		return err
	}

	err = i.Insert(rowID, updatedRow)

	if err != nil {
		return err
	}

	return nil
}
