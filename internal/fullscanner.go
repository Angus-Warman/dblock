package internal

type FullScanner struct {
	cursor  *Cursor
	columns []string
}

// Columns implements [Scanner].
func (s *FullScanner) Columns() []string {
	return s.columns
}

// Next implements [Scanner].
func (s *FullScanner) Next() (key []byte, row Row, ok bool, err error) {
	var zero Row

	k, v, ok, err := s.cursor.Next()

	if !ok || err != nil {
		return nil, zero, ok, err
	}

	row, err = DecodeRow(v)
	return k, row, true, nil
}

func NewFullScanner(tree *Tree, columns []string) (Scanner, error) {
	start, end, err := tree.KeyRange()

	if err != nil {
		return nil, err
	}

	cursor := tree.NewCursor(start, end)

	return &FullScanner{
		cursor:  cursor,
		columns: columns,
	}, nil
}
