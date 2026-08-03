package internal

import "dblock2/internal/pragma"

// PragmaScanner yields a single row holding one pragma value.
type PragmaScanner struct {
	columns []string
	value   any
	done    bool
}

func NewPragmaScanner(key pragma.Property, value any) Scanner {
	return &PragmaScanner{
		columns: []string{string(key)},
		value:   value,
	}
}

// Columns implements [Scanner].
func (s *PragmaScanner) Columns() []string {
	return s.columns
}

// Next implements [Scanner].
func (s *PragmaScanner) Next() (key []byte, row Row, ok bool, err error) {
	if s.done {
		return nil, Row{}, false, nil
	}

	s.done = true

	return nil, Row{Values: []any{s.value}}, true, nil
}
