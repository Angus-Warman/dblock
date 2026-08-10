package internal

type IndexScanner struct {
	index     *Index
	table     *Table
	idxCursor *Cursor
	tableTree *Tree
}

func NewIndexScanner(index *Index, tableTree *Tree, table *Table) (*IndexScanner, error) {
	first, last, err := index.idxTree.KeyRange()

	if err != nil {
		return nil, err
	}

	idxCursor := index.idxTree.NewCursor(first, last)

	return &IndexScanner{
		index:     index,
		table:     table,
		idxCursor: idxCursor,
		tableTree: tableTree,
	}, nil
}

func (s *IndexScanner) Next() ([]byte, Row, bool, error) {
	_, encodedRowID, ok, err := s.idxCursor.Next()

	if !ok || err != nil {
		return nil, Row{}, ok, err
	}

	encodedRow, found, err := s.tableTree.Search(encodedRowID)

	if !found || err != nil {
		return nil, Row{}, found, err
	}

	row, err := DecodeRow(encodedRow)

	if err != nil {
		return nil, Row{}, false, err
	}

	return encodedRowID, row, true, nil
}

func (s *IndexScanner) Columns() []string {
	return s.table.ColumnNames()
}
