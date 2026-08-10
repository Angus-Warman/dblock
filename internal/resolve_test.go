package internal

import (
	"testing"

	"github.com/Angus-Warman/dblock/internal/parser"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestToAny(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name     string
		value    parser.Value
		expected any
	}{
		{name: "arg", value: parser.Value{Arg: "?"}, expected: "?"},
		{name: "single quoted string", value: parser.Value{Str: "'bar'"}, expected: "bar"},
		{name: "double quoted string", value: parser.Value{Str: `"bar"`}, expected: "bar"},
		{name: "backtick string", value: parser.Value{Str: "`bar`"}, expected: "bar"},
		{name: "empty string", value: parser.Value{Str: "''"}, expected: ""},
		{name: "integer", value: parser.Value{Num: "42"}, expected: int64(42)},
		{name: "real", value: parser.Value{Num: "3.14"}, expected: float64(3.14)},
		{name: "bytes", value: parser.Value{Bytes: "0xDEADBEEF"}, expected: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{name: "true", value: parser.Value{Bool: "TRUE"}, expected: true},
		{name: "false", value: parser.Value{Bool: "FALSE"}, expected: false},
		{name: "null", value: parser.Value{Null: "NULL"}, expected: nil},
		{name: "uuid", value: parser.Value{Uuid: validUUID.String()}, expected: validUUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toAny(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestToAnyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		value parser.Value
	}{
		{name: "bad number", value: parser.Value{Num: "abc"}},
		{name: "bad hex", value: parser.Value{Bytes: "0x1"}},
		{name: "bad bool", value: parser.Value{Bool: "MAYBE"}},
		{name: "bad uuid", value: parser.Value{Uuid: "not-a-uuid"}},
		{name: "empty value", value: parser.Value{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := toAny(tt.value)
			require.Error(t, err)
		})
	}
}

func mustParse(t *testing.T, query string) *parser.ParsedStmt {
	t.Helper()
	p, err := parser.Parse(query)
	require.NoError(t, err)
	return p
}

func mustSelect(t *testing.T, query string) *SelectStmt {
	t.Helper()
	parsed := mustParse(t, query)
	require.NotNil(t, parsed.Select)
	stmt, err := resolveSelect(parsed.Select)
	require.NoError(t, err)
	require.NotNil(t, stmt.selectStmt)
	return stmt.selectStmt
}

func mustCreateIndex(t *testing.T, query string) *CreateIdxStmt {
	t.Helper()
	parsed := mustParse(t, query)
	require.NotNil(t, parsed.CreateIdx)
	stmt, err := resolveCreateIdx(parsed.CreateIdx)
	require.NoError(t, err)
	require.NotNil(t, stmt.createIdxStmt)
	return stmt.createIdxStmt
}

func TestResolveCreateIndex(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		idxName     string
		tableName   string
		columnNames []string
		unique      bool
	}{
		{
			name:        "single column",
			query:       "CREATE INDEX bar ON foo (id)",
			idxName:     "bar",
			tableName:   "foo",
			columnNames: []string{"id"},
		},
		{
			name:        "multiple columns",
			query:       "CREATE INDEX bar ON foo (id, created)",
			idxName:     "bar",
			tableName:   "foo",
			columnNames: []string{"id", "created"},
		},
		{
			name:        "if not exists",
			query:       "CREATE INDEX IF NOT EXISTS bar ON foo (id, created)",
			idxName:     "bar",
			tableName:   "foo",
			columnNames: []string{"id", "created"},
		},
		{
			name:        "unique",
			query:       "CREATE UNIQUE INDEX IF NOT EXISTS bar ON foo (id)",
			idxName:     "bar",
			tableName:   "foo",
			columnNames: []string{"id"},
			unique:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := mustCreateIndex(t, tt.query)
			require.Equal(t, tt.idxName, idx.idxName)
			require.Equal(t, tt.tableName, idx.tableName)
			require.Equal(t, tt.columnNames, idx.columnNames)
			require.Equal(t, tt.unique, idx.unique)
		})
	}
}

func TestResolveCreateIndexIfNotExists(t *testing.T) {
	parsed := mustParse(t, "CREATE INDEX IF NOT EXISTS bar ON foo (id)")
	require.NotNil(t, parsed.CreateIdx)
	require.NotNil(t, parsed.CreateIdx.ExistClause)

	parsed = mustParse(t, "CREATE INDEX bar ON foo (id)")
	require.NotNil(t, parsed.CreateIdx)
	require.Nil(t, parsed.CreateIdx.ExistClause)
}

func TestResolveTableNameAlias(t *testing.T) {
	s := mustSelect(t, "SELECT * FROM foo f")
	require.Equal(t, s.tableName, "foo")
	s = mustSelect(t, "SELECT * FROM foo AS f")
	require.Equal(t, s.tableName, "foo")
}

func TestResolveColumnAlias(t *testing.T) {
	s := mustSelect(t, "SELECT * FROM foo AS f ORDER BY f.bar")
	require.Equal(t, s.tableName, "foo")
}

func TestResolveJoins(t *testing.T) {
	type testcase struct {
		query string
		mode  JoinMode
	}

	testcases := []testcase{
		{
			query: "SELECT * FROM foo JOIN bar ON foo.id = bar.id",
			mode:  InnerJoin,
		},
		{
			query: "SELECT * FROM foo INNER JOIN bar ON foo.id = bar.id",
			mode:  InnerJoin,
		},
		{
			query: "SELECT * FROM foo LEFT JOIN bar ON foo.id = bar.id",
			mode:  LeftOuterJoin,
		},
		{
			query: "SELECT * FROM foo LEFT OUTER JOIN bar ON foo.id = bar.id",
			mode:  LeftOuterJoin,
		},
		{
			query: "SELECT * FROM foo RIGHT JOIN bar ON foo.id = bar.id",
			mode:  RightOuterJoin,
		},
		{
			query: "SELECT * FROM foo RIGHT OUTER JOIN bar ON foo.id = bar.id",
			mode:  RightOuterJoin,
		},
		{
			query: "SELECT * FROM foo FULL JOIN bar ON foo.id = bar.id",
			mode:  FullOuterJoin,
		},
		{
			query: "SELECT * FROM foo FULL OUTER JOIN bar ON foo.id = bar.id",
			mode:  FullOuterJoin,
		},
		{
			query: "SELECT * FROM foo CROSS JOIN bar ON foo.id = bar.id",
			mode:  CrossJoin,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.query, func(t *testing.T) {
			s := mustSelect(t, tc.query)
			require.Equal(t, s.tableName, "foo")
			require.Len(t, s.joins, 1)
			require.Equal(t, s.joins[0].mode, tc.mode)
		})
	}
}
