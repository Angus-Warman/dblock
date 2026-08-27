package internal

import (
	"testing"

	"uuid"

	"github.com/stretchr/testify/require"
)

func TestEvalLiterals(t *testing.T) {
	tests := []struct {
		name string
		expr Expr
		want any
	}{
		{name: "int", expr: Expr{Kind: IntExpr, Int: 42}, want: int64(42)},
		{name: "real", expr: Expr{Kind: RealExpr, Real: 3.5}, want: 3.5},
		{name: "text", expr: Expr{Kind: TextExpr, Text: "hi"}, want: "hi"},
		{name: "arg", expr: Expr{Kind: ArgExpr}, want: "sure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalExpr(&tt.expr, nil, Row{}, []any{"sure"})
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEvalColumn(t *testing.T) {
	expr := &Expr{Kind: ColumnExpr, Column: "a"}
	got, err := evalExpr(expr, []string{"a", "b"}, Row{Values: []any{3.5, 4.5}}, nil)
	require.NoError(t, err)
	require.Equal(t, 3.5, got)
}

func TestEvalColumnUnknown(t *testing.T) {
	expr := &Expr{Kind: ColumnExpr, Column: "nope"}
	_, err := evalExpr(expr, []string{"a"}, Row{Values: []any{1}}, nil)
	require.Error(t, err)
}

func TestEvalBinaryArithmetic(t *testing.T) {
	tests := []struct {
		name  string
		left  any
		op    Operator
		right any
		want  any
	}{
		{name: "float multiply", left: 3.5, op: Multiply, right: 4.5, want: 15.75},
		{name: "int add", left: int64(2), op: Add, right: int64(3), want: int64(5)},
		{name: "mixed add", left: int64(2), op: Add, right: 3.5, want: 5.5},
		{name: "int divide", left: int64(7), op: Divide, right: int64(2), want: int64(3)},
		{name: "float subtract", left: 10.0, op: Subtract, right: 2.5, want: 7.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalBinary(tt.op, tt.left, tt.right)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEvalBinaryDivideByZero(t *testing.T) {
	_, err := evalBinary(Divide, int64(1), int64(0))
	require.Error(t, err)
}

func TestEvalBinaryNull(t *testing.T) {
	got, err := evalBinary(Multiply, nil, 4.5)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestEvalBinaryComparison(t *testing.T) {
	tests := []struct {
		name  string
		left  any
		op    Operator
		right any
		want  any
	}{
		{name: "real less", left: 3.5, op: LessThan, right: 4.5, want: true},
		{name: "mixed equal", left: int64(4), op: Equal, right: 4.0, want: true},
		{name: "text equal", left: "a", op: Equal, right: "a", want: true},
		{name: "text not equal", left: "a", op: NotEqual, right: "b", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalBinary(tt.op, tt.left, tt.right)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEvalBinaryLogical(t *testing.T) {
	got, err := evalBinary(And, true, false)
	require.NoError(t, err)
	require.Equal(t, false, got)

	got, err = evalBinary(Or, true, false)
	require.NoError(t, err)
	require.Equal(t, true, got)
}

func TestEvalBinaryUnknownOp(t *testing.T) {
	_, err := evalBinary("~", int64(1), int64(2))
	require.Error(t, err)
}

func TestEvalResolvedExpression(t *testing.T) {
	p := mustParse(t, "SELECT a * b FROM foo")
	expr := p.Select.List.Items[0].Expr

	resolved, err := NewExprMachine().resolveExpr(expr)
	require.NoError(t, err)

	got, err := evalExpr(resolved, []string{"a", "b"}, Row{Values: []any{3.5, 4.5}}, nil)
	require.NoError(t, err)
	require.Equal(t, 15.75, got)
}

func TestEvalUuidFunc(t *testing.T) {
	expr := &Expr{Kind: FuncKind, FuncCall: &FuncExpr{Name: UuidFunc}}
	got, err := evalFunc(expr, Row{}, nil)
	require.NoError(t, err)

	id, ok := got.(uuid.UUID)
	require.True(t, ok, "UUID() should return a uuid.UUID, got %T", got)
	require.NotEqual(t, uuid.Nil(), id)
}

func TestEvalRowIdFunc(t *testing.T) {
	expr := &Expr{Kind: FuncKind, FuncCall: &FuncExpr{Name: RowIdFunc}}
	got, err := evalFunc(expr, Row{ID: 7}, nil)
	require.NoError(t, err)
	require.Equal(t, int64(7), got)
}

func TestEvalRowIdFuncNoContext(t *testing.T) {
	expr := &Expr{Kind: FuncKind, FuncCall: &FuncExpr{Name: RowIdFunc}}
	_, err := evalFunc(expr, Row{}, nil)
	require.Error(t, err)
}

func TestEvalUnsupportedFunc(t *testing.T) {
	expr := &Expr{Kind: FuncKind, FuncCall: &FuncExpr{Name: "NOPE"}}
	_, err := evalFunc(expr, Row{}, nil)
	require.Error(t, err)
}

func TestResolveRowIdAndUuidFuncs(t *testing.T) {
	for _, name := range []string{"ROWID", "NEW_UUID"} {
		p := mustParse(t, "SELECT "+name+"() FROM foo")
		expr, err := NewExprMachine().resolveExpr(p.Select.List.Items[0].Expr)
		require.NoError(t, err)
		require.Equal(t, FuncKind, expr.Kind)
		require.Equal(t, FuncName(name), expr.FuncCall.Name)
		require.Empty(t, expr.FuncCall.Args)
	}
}

func TestResolveRowIdFuncWithArgsRejected(t *testing.T) {
	p := mustParse(t, "SELECT ROWID(1) FROM foo")
	_, err := NewExprMachine().resolveExpr(p.Select.List.Items[0].Expr)
	require.Error(t, err)
}

func TestResolveProjectionExpression(t *testing.T) {
	p := mustParse(t, "SELECT a * b FROM foo")

	proj, err := NewExprMachine().resolveProjection(p.Select.List.Items)
	require.NoError(t, err)
	require.Len(t, proj, 1)
	require.NotNil(t, proj[0].expr)
}

func TestResolveProjectionPlainColumn(t *testing.T) {
	p := mustParse(t, "SELECT a FROM foo")

	proj, err := NewExprMachine().resolveProjection(p.Select.List.Items)
	require.NoError(t, err)
	require.Len(t, proj, 1)
	require.Nil(t, proj[0].expr)
	require.Equal(t, "a", proj[0].source)
}

func TestExprString(t *testing.T) {
	p := mustParse(t, "SELECT a * b FROM foo")

	resolved, err := NewExprMachine().resolveExpr(p.Select.List.Items[0].Expr)
	require.NoError(t, err)

	require.Equal(t, "(a * b)", exprString(resolved))
}
