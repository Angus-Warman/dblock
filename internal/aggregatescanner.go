package internal

import (
	"fmt"
	"strings"
)

type passThrough struct {
	expr *Expr
	name string
}

type funcGroup struct {
	accumulators []Accumulator
	ptValues     []any
}

// AggregateScanner collapses the rows from a base scanner into one output row per
// group, evaluating aggregate function calls (COUNT, MIN, MAX, SUM, AVG) in
// the projection. Projection items that are not aggregate calls are passed
// through, taking their value from the first row of each group (which is
// exact for GROUP BY columns).
type AggregateScanner struct {
	baseColumns  []string
	funcCalls    []*Expr
	passThroughs []passThrough
	columns      []string
	groupExprs   []*Expr
	ungrouped    bool

	groups []funcGroup

	// outIsFunc/outIdx map each output column (in projection order) back to
	// either its accumulator or pass-through slot.
	outIsFunc []bool
	outIdx    []int

	nextGroup int
	args      []any
}

func NewAggregateScanner(base Scanner, stmt *SelectStmt, args []any) (*AggregateScanner, error) {
	if base == nil {
		return nil, fmt.Errorf("func: base scanner is nil")
	}

	if stmt == nil {
		return nil, fmt.Errorf("func: select stmt is nil")
	}

	baseColumns := base.Columns()

	funcCalls := []*Expr{}
	passThroughs := []passThrough{}
	columns := []string{}
	outIsFunc := []bool{}
	outIdx := []int{}

	for _, item := range stmt.projection {
		if item.expr != nil && item.expr.Kind == FuncKind {
			funcCalls = append(funcCalls, item.expr)

			name := exprString(item.expr)
			if item.alias != "" {
				name = item.alias
			}

			columns = append(columns, name)
			outIsFunc = append(outIsFunc, true)
			outIdx = append(outIdx, len(funcCalls)-1)
			continue
		}

		pt := passThrough{}
		if item.expr != nil {
			pt.expr = item.expr
			pt.name = exprString(item.expr)
		} else {
			pt.expr = &Expr{Kind: ColumnExpr, Column: item.source}
			pt.name = item.source
		}

		if item.alias != "" {
			pt.name = item.alias
		}

		passThroughs = append(passThroughs, pt)

		columns = append(columns, pt.name)
		outIsFunc = append(outIsFunc, false)
		outIdx = append(outIdx, len(passThroughs)-1)
	}

	if len(funcCalls) == 0 && len(passThroughs) == 0 {
		return nil, fmt.Errorf("func: projection has no aggregate or group columns")
	}

	s := &AggregateScanner{
		baseColumns:  baseColumns,
		funcCalls:    funcCalls,
		passThroughs: passThroughs,
		columns:      columns,
		groupExprs:   stmt.groupBy,
		ungrouped:    len(stmt.groupBy) == 0,
		outIsFunc:    outIsFunc,
		outIdx:       outIdx,
		args:         args,
	}

	if err := s.bufferGroups(base); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *AggregateScanner) Columns() []string {
	return s.columns
}

func (s *AggregateScanner) Next() (key []byte, row Row, ok bool, err error) {
	if s.nextGroup >= len(s.groups) {
		return nil, Row{}, false, nil
	}

	out := s.buildRow(&s.groups[s.nextGroup])
	s.nextGroup++

	return nil, out, true, nil
}

// bufferGroups reads the entire base scanner and splits it into groups by the
// GROUP BY key, preserving first-seen order.
func (s *AggregateScanner) bufferGroups(base Scanner) error {
	byKey := map[string]int{}

	for {
		_, row, ok, err := base.Next()
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		key, err := s.groupKey(row)
		if err != nil {
			return err
		}

		gi, seen := byKey[key]
		if !seen {
			gi = len(s.groups)
			byKey[key] = gi

			g := funcGroup{
				accumulators: make([]Accumulator, len(s.funcCalls)),
				ptValues:     make([]any, len(s.passThroughs)),
			}

			for i, fc := range s.funcCalls {
				g.accumulators[i] = newAccumulator(fc.FuncCall.Name)
			}

			for i, pt := range s.passThroughs {
				val, err := evalExpr(pt.expr, s.baseColumns, row.Values, s.args)
				if err != nil {
					return fmt.Errorf("func: pass-through %q: %w", pt.name, err)
				}

				g.ptValues[i] = val
			}

			s.groups = append(s.groups, g)
		}

		if err := s.accumulate(&s.groups[gi], row); err != nil {
			return err
		}
	}

	// An ungrouped aggregate over an empty input still yields one row
	// (e.g. COUNT(*) over an empty table is 0).
	if len(s.groups) == 0 && s.ungrouped {
		g := funcGroup{
			accumulators: make([]Accumulator, len(s.funcCalls)),
			ptValues:     make([]any, len(s.passThroughs)),
		}

		for i, fc := range s.funcCalls {
			g.accumulators[i] = newAccumulator(fc.FuncCall.Name)
		}

		s.groups = append(s.groups, g)
	}

	return nil
}

func (s *AggregateScanner) groupKey(row Row) (string, error) {
	if len(s.groupExprs) == 0 {
		return "", nil
	}

	parts := make([]string, len(s.groupExprs))

	for i, expr := range s.groupExprs {
		val, err := evalExpr(expr, s.baseColumns, row.Values, s.args)
		if err != nil {
			return "", fmt.Errorf("func: group by: %w", err)
		}

		parts[i] = fmt.Sprint(val)
	}

	return strings.Join(parts, "\x00"), nil
}

func (s *AggregateScanner) accumulate(g *funcGroup, row Row) error {
	for i, fc := range s.funcCalls {
		acc := g.accumulators[i]
		fun := fc.FuncCall

		if fun.Name == CountFunc && (len(fun.Args) == 0 || fun.Args[0].Star) {
			// COUNT(*) / COUNT() counts every row unconditionally.
			acc.Update(true)
			continue
		}

		if len(fun.Args) == 0 {
			return fmt.Errorf("func: %s requires an argument", fun.Name)
		}

		val, err := evalExpr(&fun.Args[0], s.baseColumns, row.Values, s.args)
		if err != nil {
			return fmt.Errorf("func: %s: %w", fun.Name, err)
		}

		acc.Update(val)
	}

	return nil
}

func (s *AggregateScanner) buildRow(g *funcGroup) Row {
	rvs := make([]any, len(s.outIsFunc))

	for i := range rvs {
		if s.outIsFunc[i] {
			rvs[i] = g.accumulators[s.outIdx[i]].Result()
		} else {
			rvs[i] = g.ptValues[s.outIdx[i]]
		}
	}

	return Row{Values: rvs}
}
