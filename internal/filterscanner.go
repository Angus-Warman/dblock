package internal

import (
	"fmt"
)

// FilterScanner filters rows from a base scanner by evaluating a resolved
// WHERE expression per row, keeping only rows where it evaluates to true.
type FilterScanner struct {
	base Scanner
	expr *Expr
	args []any
}

// Columns implements [Scanner].
func (w *FilterScanner) Columns() []string {
	return w.base.Columns()
}

// Next implements [Scanner].
func (w *FilterScanner) Next() (key []byte, row Row, ok bool, err error) {
	for {
		key, row, ok, err = w.base.Next()

		if !ok || err != nil {
			return nil, Row{}, ok, err
		}

		val, err := evalExpr(w.expr, w.base.Columns(), row.Values, w.args)

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

func NewFilterScanner(base Scanner, expr *Expr, args []any) (Scanner, error) {
	if base == nil {
		return nil, fmt.Errorf("where: base scanner is nil")
	}

	if expr == nil {
		return nil, fmt.Errorf("where: expression is nil")
	}

	return &FilterScanner{
		base: base,
		expr: expr,
		args: args,
	}, nil
}
