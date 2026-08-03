package internal

import (
	"bytes"
	"fmt"
	"time"
)

type joinEntry struct {
	key []byte
	row Row
}

type JoinScanner struct {
	left         Scanner
	right        Scanner
	columns      []string
	leftJoinIdx  int
	rightJoinIdx int
	rightRows    []joinEntry
	rightIdx     int
	currentLeft  joinEntry
	needsNewLeft bool
}

func NewJoinScanner(left, right Scanner, stmt *SelectStmt, join *JoinStmt) (*JoinScanner, error) {
	leftCols := left.Columns()
	rightCols := right.Columns()

	leftJoinIdx := -1
	for i, col := range leftCols {
		if col == join.on.left.column {
			leftJoinIdx = i
			break
		}
	}

	rightJoinIdx := -1
	for i, col := range rightCols {
		if col == join.on.right.column {
			rightJoinIdx = i
			break
		}
	}

	if leftJoinIdx < 0 || rightJoinIdx < 0 {
		return nil, fmt.Errorf("join column not found")
	}

	columns := make([]string, 0, len(leftCols)+len(rightCols)-1)

	for i, col := range leftCols {
		if i == leftJoinIdx {
			columns = append(columns, col)
			continue
		}
		columns = append(columns, stmt.tableName+"."+col)
	}

	for i, col := range rightCols {
		if i == rightJoinIdx {
			continue
		}
		columns = append(columns, join.tableName+"."+col)
	}

	return &JoinScanner{
		left:         left,
		right:        right,
		columns:      columns,
		leftJoinIdx:  leftJoinIdx,
		rightJoinIdx: rightJoinIdx,
		needsNewLeft: true,
	}, nil
}

// Columns implements [Scanner].
func (j *JoinScanner) Columns() []string {
	return j.columns
}

// Next implements [Scanner].
func (j *JoinScanner) Next() (key []byte, row Row, ok bool, err error) {
	if err := j.loadRightRows(); err != nil {
		return nil, Row{}, false, err
	}

	for {
		if j.needsNewLeft || j.rightIdx >= len(j.rightRows) {
			key, row, ok, err := j.left.Next()

			if err != nil {
				return nil, Row{}, false, err
			}

			if !ok {
				return nil, Row{}, false, nil
			}

			j.currentLeft = joinEntry{key: key, row: row}
			j.rightIdx = 0
			j.needsNewLeft = false
		}

		for j.rightIdx < len(j.rightRows) {
			rightEntry := j.rightRows[j.rightIdx]
			j.rightIdx++

			leftVal := j.currentLeft.row.Values[j.leftJoinIdx]
			rightVal := rightEntry.row.Values[j.rightJoinIdx]

			if !valuesEqual(leftVal, rightVal) {
				continue
			}

			return j.currentLeft.key, combineRows(j.currentLeft.row, rightEntry.row, j.rightJoinIdx), true, nil
		}

		j.needsNewLeft = true
	}
}

func (j *JoinScanner) loadRightRows() error {
	if j.rightRows != nil {
		return nil
	}

	for {
		_, row, ok, err := j.right.Next()

		if err != nil {
			return err
		}

		if !ok {
			break
		}

		j.rightRows = append(j.rightRows, joinEntry{row: row})
	}

	return nil
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case []byte:
		bv, ok := b.([]byte)
		if !ok {
			return false
		}
		return bytes.Equal(av, bv)
	case time.Time:
		bv, ok := b.(time.Time)
		if !ok {
			return false
		}
		return av.Equal(bv)
	default:
		return a == b
	}
}

func combineRows(left, right Row, rightJoinIdx int) Row {
	vals := make([]any, 0, len(left.Values)+len(right.Values)-1)
	vals = append(vals, left.Values...)

	for i, v := range right.Values {
		if i == rightJoinIdx {
			continue
		}
		vals = append(vals, v)
	}

	return Row{Values: vals}
}
