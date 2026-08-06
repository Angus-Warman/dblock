package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveExpr(t *testing.T) {
	p := mustParse(t, "SELECT 2 + 3 / 4 - a FROM foo")
	require.Len(t, p.Select.List.Items, 1)
	expr := p.Select.List.Items[0].Expr
	require.NotNil(t, expr)
	require.Len(t, expr.Factors, 7)

	expected := &Expr{
		Kind: BinaryKind,
		Binary: &BinaryExpr{
			Left: &Expr{
				Kind: BinaryKind,
				Binary: &BinaryExpr{
					Left: &Expr{
						Kind: IntExpr,
						Int:  2,
					},
					Op: Add,
					Right: &Expr{
						Kind: BinaryKind,
						Binary: &BinaryExpr{
							Op: Divide,
							Left: &Expr{
								Kind: IntExpr,
								Int:  3,
							},
							Right: &Expr{
								Kind: IntExpr,
								Int:  4,
							},
						},
					},
				},
			},
			Op: Subtract,
			Right: &Expr{
				Kind:   ColumnExpr,
				Column: "a",
			},
		},
	}

	actual, err := NewExprMachine().resolveExpr(expr)

	require.NoError(t, err)

	require.Equal(t, expected, actual)
}

func TestResolveMultiply(t *testing.T) {
	p := mustParse(t, "SELECT a * 4 FROM foo")
	require.Len(t, p.Select.List.Items, 1)
	expr := p.Select.List.Items[0].Expr
	require.NotNil(t, expr)
	require.Len(t, expr.Factors, 3)

	expected := &Expr{
		Kind: BinaryKind,
		Binary: &BinaryExpr{
			Left: &Expr{
				Kind:   ColumnExpr,
				Column: "a",
			},
			Op: Multiply,
			Right: &Expr{
				Kind: IntExpr,
				Int:  4,
			},
		},
	}

	actual, err := NewExprMachine().resolveExpr(expr)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
