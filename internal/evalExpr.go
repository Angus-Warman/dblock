package internal

import (
	"fmt"
	"strconv"
	"strings"
)

// evalExpr evaluates a resolved expression tree against a single row. columns
// names each value in values (the base scanner's column list), so column
// references can be resolved by name.
func evalExpr(expr *Expr, columns []string, values []any, args []any) (any, error) {
	if expr == nil {
		return nil, fmt.Errorf("eval: nil expression")
	}

	switch expr.Kind {
	case IntExpr:
		return expr.Int, nil

	case RealExpr:
		return expr.Real, nil

	case BlobExpr:
		return expr.Blob, nil

	case TextExpr:
		return expr.Text, nil

	case ArgExpr:
		if expr.ArgIndex >= len(args) {
			return nil, fmt.Errorf("eval: missing argument for placeholder %d", expr.ArgIndex+1)
		}

		return args[expr.ArgIndex], nil

	case ColumnExpr:
		idx := findColumn(columns, expr.Column)

		if idx < 0 {
			return nil, fmt.Errorf("eval: no such column %q", expr.Column)
		}

		return values[idx], nil

	case BinaryKind:
		left, err := evalExpr(expr.Binary.Left, columns, values, args)

		if err != nil {
			return nil, err
		}

		right, err := evalExpr(expr.Binary.Right, columns, values, args)

		if err != nil {
			return nil, err
		}

		return evalBinary(expr.Binary.Op, left, right)

	case FuncKind:
		return nil, fmt.Errorf("eval: function calls not implemented")

	default:
		return nil, fmt.Errorf("eval: unsupported expression kind %d", expr.Kind)
	}
}

func findColumn(columns []string, name string) int {
	for i, c := range columns {
		if c == name {
			return i
		}
	}

	return -1
}

// evalBinary applies a binary operator to two already-evaluated operands.
// Per SQL semantics, any operation on a NULL operand yields NULL.
func evalBinary(op Operator, left, right any) (any, error) {
	if left == nil || right == nil {
		return nil, nil
	}

	switch op {
	case Add, Subtract, Multiply, Divide, Modulo:
		return evalArithmetic(op, left, right)

	case Equal, NotEqual, LessThan, LessThanOrEq, GreaterThan, GreaterThanOrEq:
		return evalComparison(op, left, right)

	case And, Or:
		return evalLogical(op, left, right)
	}

	return nil, fmt.Errorf("eval: unsupported operator %q", op)
}

// evalArithmetic evaluates numeric operators. Integer operands stay integers
// when both sides are int64; any float operand promotes the whole operation.
func evalArithmetic(op Operator, left, right any) (any, error) {
	if li, ok := left.(int64); ok {
		if ri, ok := right.(int64); ok {
			switch op {
			case Add:
				return li + ri, nil
			case Subtract:
				return li - ri, nil
			case Multiply:
				return li * ri, nil
			case Divide:
				if ri == 0 {
					return nil, fmt.Errorf("eval: division by zero")
				}
				return li / ri, nil
			case Modulo:
				if ri == 0 {
					return nil, fmt.Errorf("eval: division by zero")
				}
				return li % ri, nil
			}
		}
	}

	lf, ok := toFloat(left)

	if !ok {
		return nil, fmt.Errorf("eval: cannot apply %q to %T", op, left)
	}

	rf, ok := toFloat(right)

	if !ok {
		return nil, fmt.Errorf("eval: cannot apply %q to %T", op, right)
	}

	switch op {
	case Add:
		return lf + rf, nil
	case Subtract:
		return lf - rf, nil
	case Multiply:
		return lf * rf, nil
	case Divide:
		if rf == 0 {
			return nil, fmt.Errorf("eval: division by zero")
		}
		return lf / rf, nil
	case Modulo:
		return float64(int64(lf) % int64(rf)), nil
	}

	return nil, fmt.Errorf("eval: unsupported operator %q", op)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}

	return 0, false
}

// evalComparison evaluates comparison operators. Numeric operands compare by
// value regardless of int/float type; everything else falls back to the
// storage-layer ordering used by ORDER BY.
func evalComparison(op Operator, left, right any) (any, error) {
	var cmp int

	lf, lOK := toFloat(left)
	rf, rOK := toFloat(right)

	switch {
	case lOK && rOK:
		switch {
		case lf < rf:
			cmp = -1
		case lf > rf:
			cmp = 1
		}

	case !lOK && !rOK:
		cmp = compareValues(left, right)

	default:
		return nil, fmt.Errorf("eval: cannot compare %T and %T", left, right)
	}

	switch op {
	case Equal:
		return cmp == 0, nil
	case NotEqual:
		return cmp != 0, nil
	case LessThan:
		return cmp < 0, nil
	case LessThanOrEq:
		return cmp <= 0, nil
	case GreaterThan:
		return cmp > 0, nil
	case GreaterThanOrEq:
		return cmp >= 0, nil
	}

	return nil, fmt.Errorf("eval: unsupported operator %q", op)
}

func evalLogical(op Operator, left, right any) (any, error) {
	lb, ok := left.(bool)

	if !ok {
		return nil, fmt.Errorf("eval: cannot apply %q to %T", op, left)
	}

	rb, ok := right.(bool)

	if !ok {
		return nil, fmt.Errorf("eval: cannot apply %q to %T", op, right)
	}

	switch op {
	case And:
		return lb && rb, nil
	case Or:
		return lb || rb, nil
	}

	return nil, fmt.Errorf("eval: unsupported operator %q", op)
}

// exprString renders a resolved expression back to SQL text, used as the
// default output column name for computed projections.
func exprString(expr *Expr) string {
	if expr == nil {
		return ""
	}

	switch expr.Kind {
	case StarExpr:
		return "*"

	case IntExpr:
		return strconv.FormatInt(expr.Int, 10)

	case RealExpr:
		return strconv.FormatFloat(expr.Real, 'g', -1, 64)

	case BlobExpr:
		return fmt.Sprintf("x'%x'", expr.Blob)

	case TextExpr:
		return "'" + strings.ReplaceAll(expr.Text, "'", "''") + "'"

	case ColumnExpr:
		return expr.Column

	case BinaryKind:
		return "(" + exprString(expr.Binary.Left) + " " + string(expr.Binary.Op) + " " + exprString(expr.Binary.Right) + ")"

	case FuncKind:
		args := make([]string, len(expr.FuncCall.Args))

		for i := range args {
			args[i] = exprString(&expr.FuncCall.Args[i])
		}

		return string(expr.FuncCall.Name) + "(" + strings.Join(args, ", ") + ")"

	case ArgExpr:
		return "?"
	}

	return ""
}
