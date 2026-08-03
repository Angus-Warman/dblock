package internal

import (
	"fmt"
	"slices"
)

type ProjectorScanner struct {
	base       Scanner
	columnIdx  []int
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

	values := make([]any, len(p.columnIdx))

	for i, idx := range p.columnIdx {
		values[i] = fullRow.Values[idx]
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
			outColumns: columns,
		}, nil
	}

	indices := make([]int, len(stmt.projection))
	aliasedColumns := make([]string, len(stmt.projection))

	for i, proj := range stmt.projection {
		idx := slices.Index(columns, proj.source)

		if idx < 0 {
			return nil, fmt.Errorf("projector: no such column %q", proj.source)
		}

		indices[i] = idx

		if proj.alias != "" {
			aliasedColumns[i] = proj.alias
		} else {
			aliasedColumns[i] = proj.source
		}
	}

	return &ProjectorScanner{
		base:       base,
		columnIdx:  indices,
		outColumns: aliasedColumns,
	}, nil
}
