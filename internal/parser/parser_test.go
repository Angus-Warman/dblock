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
