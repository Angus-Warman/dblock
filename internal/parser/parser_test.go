package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, query string) *ParsedStmt {
	p, err := Parse(query)
	require.NoError(t, err)
	return p
}

func TestParsePragma(t *testing.T) {
	p := mustParse(t, "PRAGMA foo")
	require.NotNil(t, p)
	require.NotNil(t, p.Pragma)
	require.Equal(t, p.Pragma.Property, "foo")
}

func TestParsePragmaValue(t *testing.T) {
	p := mustParse(t, "PRAGMA foo 1")
	require.NotNil(t, p)
	require.NotNil(t, p.Pragma)
	require.Equal(t, p.Pragma.Property, "foo")
	require.Equal(t, p.Pragma.Value, "1")
}

func TestParseNegativeNumber(t *testing.T) {
	p := mustParse(t, "INSERT INTO foo VALUES (-1)")
	require.NotNil(t, p)
}

func TestParseWhere(t *testing.T) {
	p := mustParse(t, "SELECT * FROM foo WHERE a <> 1")
	require.Len(t, p.Select.Where.Expr.Factors, 3)
	require.Equal(t, p.Select.Where.Expr.Factors[0].Column, "a")
	require.Equal(t, p.Select.Where.Expr.Factors[1].Op, "<>")
	require.Equal(t, p.Select.Where.Expr.Factors[2].Num, "1")
}

func TestParseWhereComplex(t *testing.T) {
	p := mustParse(t, "SELECT * FROM foo JOIN bar ON foo.id = bar.id WHERE foo.a != 1 AND foo.b + 2 = bar.c")
	f := p.Select.Where.Expr.Factors
	require.Len(t, f, 9)
	require.Equal(t, "foo.a", f[0].Column)
	require.Equal(t, "!=", f[1].Op)
	require.Equal(t, "1", f[2].Num)
	require.Equal(t, "", f[3].Column) // Catch error
	require.Equal(t, "AND", f[3].Op)
	require.Equal(t, "foo.b", f[4].Column)
	require.Equal(t, "+", f[5].Op)
	require.Equal(t, "2", f[6].Num)
	require.Equal(t, "=", f[7].Op)
	require.Equal(t, "bar.c", f[8].Column)
}

func TestParseFunction(t *testing.T) {
	p := mustParse(t, "SELECT COUNT(*) FROM foo")
	require.Len(t, p.Select.List.Items, 1)
	item := p.Select.List.Items[0]
	require.NotNil(t, item.Expr.Factors[0].Func)
	funcCall := item.Expr.Factors[0].Func
	require.Len(t, funcCall.Args, 1)
	require.True(t, funcCall.Args[0].Factors[0].Star)
}

func TestParseParens(t *testing.T) {
	p := mustParse(t, "SELECT (a + 1) * (b + 2) FROM foo")
	require.Len(t, p.Select.List.Items, 1)
	f := p.Select.List.Items[0].Expr.Factors
	require.Len(t, f, 3)
	require.NotNil(t, f[0].SubExpr)
	require.NotNil(t, f[2].SubExpr)
}
