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
	left           Scanner
	right          Scanner
	columns        []string
	leftJoinIdx    int
	rightJoinIdx   int
	leftColCount   int
	rightColCount  int
	rightRows      []joinEntry
	rightIdx       int
	matchedRight   []bool
	rightUnmatched int
	currentLeft    joinEntry
	leftMatched    bool
	needsNewLeft   bool

	emitLeftUnmatched  bool // LEFT, FULL
	emitRightUnmatched bool // RIGHT, FULL
}

func NewJoinScanner(left, right Scanner, stmt *SelectStmt, join *JoinStmt) (*JoinScanner, error) {
	emitLeft := false
	emitRight := false
	switch join.mode {
	case LeftJoin, LeftOuterJoin:
		emitLeft = true
	case RightJoin, RightOuterJoin:
		emitRight = true
	case FullJoin, FullOuterJoin:
		emitLeft = true
		emitRight = true
	case InnerJoin:
		// both false
	default:
		return nil, fmt.Errorf("unsupported join mode: %v", join.mode)
	}

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

	columns := make([]string, 0, len(leftCols)+len(rightCols))

	for _, col := range leftCols {
		columns = append(columns, stmt.tableName+"."+col)
	}

	for _, col := range rightCols {
		columns = append(columns, join.tableName+"."+col)
	}

	return &JoinScanner{
		left:          left,
		right:         right,
		columns:       columns,
		leftJoinIdx:   leftJoinIdx,
		rightJoinIdx:  rightJoinIdx,
		leftColCount:  len(leftCols),
		rightColCount: len(rightCols),
		needsNewLeft:  true,

		emitLeftUnmatched:  emitLeft,  // LEFT, FULL
		emitRightUnmatched: emitRight, // RIGHT, FULL
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
				return j.emitUnmatchedRight()
			}

			j.currentLeft = joinEntry{key: key, row: row}
			j.rightIdx = 0
			j.leftMatched = false
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

			j.matchedRight[j.rightIdx-1] = true
			j.leftMatched = true

			return j.currentLeft.key, combineRows(j.currentLeft.row, rightEntry.row), true, nil
		}

		if !j.leftMatched && j.emitLeftUnmatched {
			return j.currentLeft.key, nullPadRight(j.currentLeft.row, j.rightColCount), true, nil
		}

		j.needsNewLeft = true
	}
}

func (j *JoinScanner) emitUnmatchedRight() (key []byte, row Row, ok bool, err error) {
	if !j.emitRightUnmatched {
		return nil, Row{}, false, nil
	}

	for j.rightUnmatched < len(j.rightRows) {
		entry := j.rightRows[j.rightUnmatched]
		j.rightUnmatched++

		if j.matchedRight[j.rightUnmatched-1] {
			continue
		}

		return entry.key, nullPadLeft(j.leftColCount, entry.row), true, nil
	}

	return nil, Row{}, false, nil
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

	j.matchedRight = make([]bool, len(j.rightRows))

	return nil
}

func valuesEqual(a, b any) bool {
	// NULLs are never equal
	if a == nil || b == nil {
		return false
	}

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

func combineRows(left, right Row) Row {
	vals := make([]any, 0, len(left.Values)+len(right.Values))
	vals = append(vals, left.Values...)
	vals = append(vals, right.Values...)

	return Row{Values: vals}
}

func nullPadRight(left Row, rightColCount int) Row {
	vals := make([]any, 0, len(left.Values)+rightColCount)
	vals = append(vals, left.Values...)

	for i := 0; i < rightColCount; i++ {
		vals = append(vals, nil)
	}

	return Row{Values: vals}
}

func nullPadLeft(leftColCount int, right Row) Row {
	vals := make([]any, leftColCount)
	vals = append(vals, right.Values...)

	return Row{Values: vals}
}
