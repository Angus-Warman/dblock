package internal

type Scanner interface {
	Next() (key []byte, row Row, ok bool, err error)
	Columns() []string
}

type FullScanner struct {
	cursor *Cursor
	pos    int64
}

// Columns implements [Scanner].
func (s *FullScanner) Columns() []string {
	return []string{"object_name", "object_type", "definition", "rootpage"}
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

func NewFullScanner(tree *Tree) (Scanner, error) {
	start, end, err := tree.KeyRange()

	if err != nil {
		return nil, err
	}

	cursor := tree.NewCursor(start, end)

	return &FullScanner{
		cursor: cursor,
	}, nil
}
