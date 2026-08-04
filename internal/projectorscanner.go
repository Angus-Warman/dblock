package internal

import (
	"fmt"
	"slices"
)

type ProjectorScanner struct {
	base       Scanner
	columnIdx  []int
	exprs      []*Expr
	outColumns []string
}

// Columns implements [Scanner].
func (p *ProjectorScanner) Columns() []string {
	return p.outColumns
}

// Next implements [Scanner].
func (p *ProjectorScanner) Next() (key []byte, row Row, ok bool, err error) {
	key, fullRow, ok, err := p.base.Next()

	if !ok || err != nil {
		return nil, Row{}, ok, err
	}

	columns := p.base.Columns()
	values := make([]any, len(p.outColumns))

	for i := range values {
		if p.exprs[i] != nil {
			values[i], err = evalExpr(p.exprs[i], columns, fullRow.Values)

			if err != nil {
				return nil, Row{}, false, err
			}

			continue
		}

		values[i] = fullRow.Values[p.columnIdx[i]]
	}

	return key, Row{Values: values}, true, nil
}

func NewProjectorScanner(base Scanner, stmt *SelectStmt) (Scanner, error) {
	if base == nil {
		return nil, fmt.Errorf("projector: base scanner is nil")
	}

	if stmt == nil {
		return nil, fmt.Errorf("projector: select stmt is nil")
	}

	columns := base.Columns()

	if len(stmt.projection) == 0 {
		indices := make([]int, len(columns))

		for i := range columns {
			indices[i] = i
		}

		return &ProjectorScanner{
			base:       base,
			columnIdx:  indices,
			exprs:      make([]*Expr, len(columns)),
			outColumns: columns,
		}, nil
	}

	columnIdx := make([]int, len(stmt.projection))
	exprs := make([]*Expr, len(stmt.projection))
	outColumns := make([]string, len(stmt.projection))

	for i, proj := range stmt.projection {
		if proj.expr != nil {
			columnIdx[i] = -1
			exprs[i] = proj.expr

			if proj.alias != "" {
				outColumns[i] = proj.alias
			} else {
				outColumns[i] = exprString(proj.expr)
			}

			continue
		}

		idx := slices.Index(columns, proj.source)

		if idx < 0 {
			return nil, fmt.Errorf("projector: no such column %q", proj.source)
		}

		columnIdx[i] = idx

		if proj.alias != "" {
			outColumns[i] = proj.alias
		} else {
			outColumns[i] = proj.source
		}
	}

	return &ProjectorScanner{
		base:       base,
		columnIdx:  columnIdx,
		exprs:      exprs,
		outColumns: outColumns,
	}, nil
}
