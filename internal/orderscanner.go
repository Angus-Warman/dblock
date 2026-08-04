package internal

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type orderEntry struct {
	key []byte
	row Row
}

type OrderScanner struct {
	base     Scanner
	columns  []string
	orderIdx []int
	rows     []orderEntry
	nextIdx  int
}

// Columns implements [Scanner].
func (s *OrderScanner) Columns() []string {
	return s.base.Columns()
}

// Next implements [Scanner].
func (s *OrderScanner) Next() (key []byte, row Row, ok bool, err error) {
	if s.nextIdx >= len(s.rows) {
		return nil, Row{}, false, nil
	}

	entry := s.rows[s.nextIdx]
	s.nextIdx++
	return entry.key, entry.row, true, nil
}

func NewOrderScanner(base Scanner, stmt *SelectStmt) (Scanner, error) {
	if base == nil {
		return nil, fmt.Errorf("order: base scanner is nil")
	}

	if stmt == nil {
		return nil, fmt.Errorf("order: select stmt is nil")
	}

	columns := base.Columns()

	orderIdx := make([]int, len(stmt.orders))

	for i, ord := range stmt.orders {
		idx := slices.Index(columns, ord.column)

		if idx < 0 && ord.table != "" {
			idx = slices.Index(columns, ord.table+"."+ord.column)
		}

		if idx < 0 {
			return nil, fmt.Errorf("order: no such column %q", ord.column)
		}

		orderIdx[i] = idx
	}

	rows := []orderEntry{}

	for {
		key, row, ok, err := base.Next()

		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}

		rows = append(rows, orderEntry{key: key, row: row})
	}

	slices.SortStableFunc(rows, func(a, b orderEntry) int {
		for _, idx := range orderIdx {
			if cmp := compareValues(a.row.Values[idx], b.row.Values[idx]); cmp != 0 {
				return cmp
			}
		}
		return 0
	})

	return &OrderScanner{
		base:     base,
		orderIdx: orderIdx,
		rows:     rows,
	}, nil
}

func compareValues(a, b any) int {
	if a == nil && b == nil {
		return 0
	}

	if a == nil {
		return -1
	}

	if b == nil {
		return 1
	}

	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}

	case int64:
		if bv, ok := b.(int64); ok {
			switch {
			case av < bv:
				return -1
			case av > bv:
				return 1
			}
			return 0
		}

	case float64:
		if bv, ok := b.(float64); ok {
			switch {
			case av < bv:
				return -1
			case av > bv:
				return 1
			}
			return 0
		}

	case bool:
		if bv, ok := b.(bool); ok {
			switch {
			case av == bv:
				return 0
			case !av:
				return -1
			}
			return 1
		}

	case []byte:
		if bv, ok := b.([]byte); ok {
			return bytes.Compare(av, bv)
		}

	case time.Time:
		if bv, ok := b.(time.Time); ok {
			switch {
			case av.Before(bv):
				return -1
			case av.After(bv):
				return 1
			}
			return 0
		}

	case uuid.UUID:
		if bv, ok := b.(uuid.UUID); ok {
			return bytes.Compare(av[:], bv[:])
		}
	}

	return 0
}
