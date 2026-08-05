package internal

import (
	"fmt"
)

type Accumulator interface {
	Update(val any) bool
	Result() any
}

func newAccumulator(name FuncName) Accumulator {
	switch name {
	case CountFunc:
		return &countAccumulator{}
	case MinFunc:
		return &minMaxAccumulator{isMin: true}
	case MaxFunc:
		return &minMaxAccumulator{isMin: false}
	case SumFunc:
		return &sumAccumulator{}
	case AvgFunc:
		return &avgAccumulator{}
	}

	panic(fmt.Errorf("no implementation for %v", name))
}

type countAccumulator struct {
	count int64
}

func (a *countAccumulator) Update(val any) bool {
	if val == nil {
		return false
	}
	a.count++
	return true
}

func (a *countAccumulator) Result() any {
	return a.count
}

type minMaxAccumulator struct {
	isMin bool
	val   any
	// TODO this should return an int if all values are int
}

func (a *minMaxAccumulator) Update(val any) bool {
	if val == nil {
		return false
	}
	if a.val == nil {
		a.val = val
		return true
	}
	cmp := compareValues(val, a.val)
	if (a.isMin && cmp < 0) || (!a.isMin && cmp > 0) {
		a.val = val
		return true
	}
	return false
}

func (a *minMaxAccumulator) Result() any {
	return a.val
}

type sumAccumulator struct {
	sum any
}

func (a *sumAccumulator) Update(val any) bool {
	if _, ok := toFloat(val); !ok {
		return false
	}

	if a.sum == nil {
		a.sum = val
		return true
	}

	sum, ok := addNumeric(a.sum, val)
	if !ok {
		return false
	}

	a.sum = sum
	return true
}

func (a *sumAccumulator) Result() any {
	if a.sum == nil {
		return 0.0
	}
	return a.sum
}

type avgAccumulator struct {
	total float64
	count int
}

func (a *avgAccumulator) Update(val any) bool {
	n, ok := toFloat(val)
	if !ok {
		return false
	}
	a.total += n
	a.count += 1
	return true
}

func (a *avgAccumulator) Result() any {
	if a.count == 0 {
		return nil
	}

	avg := a.total / float64(a.count)
	return avg
}

// addNumeric adds two numeric values, returning an int64 when both operands
// are int64 and promoting to float64 when either operand is a float.
func addNumeric(a, b any) (any, bool) {
	if ai, ok := a.(int64); ok {
		if bi, ok := b.(int64); ok {
			return ai + bi, true
		}
	}

	af, ok := toFloat(a)
	if !ok {
		return nil, false
	}

	bf, ok := toFloat(b)
	if !ok {
		return nil, false
	}

	return af + bf, true
}
