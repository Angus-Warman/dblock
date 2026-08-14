package internal

import (
	"fmt"
)

// FilterScanner filters rows from a base scanner by evaluating a resolved
// WHERE expression per row, keeping only rows where it evaluates to true.
type FilterScanner struct {
	base  Scanner
	where *Expr
	args  []any
}

func NewFilterScanner(base Scanner, where *Expr, args []any) (Scanner, error) {
	if base == nil {
		return nil, fmt.Errorf("where: base scanner is nil")
	}

	if where == nil {
		return nil, fmt.Errorf("where: expression is nil")
	}

	return &FilterScanner{
		base:  base,
		where: where,
		args:  args,
	}, nil
}

// Columns implements [Scanner].
func (s *FilterScanner) Columns() []string {
	return s.base.Columns()
}

// Next implements [Scanner].
func (s *FilterScanner) Next() (key []byte, row Row, ok bool, err error) {
	for {
		key, row, ok, err = s.base.Next()

		if !ok || err != nil {
			return nil, Row{}, ok, err
		}

		val, err := evalExpr(s.where, s.base.Columns(), row, s.args)

		if err != nil {
			return nil, Row{}, false, err
		}

		matches, isBool := val.(bool)

		if !isBool {
			return nil, Row{}, false, fmt.Errorf("where: expected boolean expression, got %T", val)
		}

		if matches {
			return key, row, true, nil
		}
	}
}
