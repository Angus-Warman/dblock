package internal

type CustomScanner struct {
	columns []string
	rows    [][]any
	idx     int
	done    bool
}

func NewCustomScanner(columns []string, values [][]any) Scanner {
	return &CustomScanner{
		columns: columns,
		rows:    values,
	}
}

// Columns implements [Scanner].
func (s *CustomScanner) Columns() []string {
	return s.columns
}

// Next implements [Scanner].
func (s *CustomScanner) Next() (key []byte, row Row, ok bool, err error) {
	if s.idx >= len(s.rows) {
		return nil, Row{}, false, nil
	}

	s.idx++

	return nil, Row{Values: s.rows[s.idx-1]}, true, nil
}
