package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectWhereMultipleArgs(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 1)")

	assertQueryColumnArgs(t, db, "SELECT b FROM foo WHERE a = ?", []any{"x"}, []any{int64(1), int64(2)})
	assertQueryColumnArgs(t, db, "SELECT a FROM foo WHERE b = ?", []any{int64(2)}, []any{"x"})
}

func TestSelectWhereAndArgs(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 2)")

	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE a = ? AND b = ?", []any{"x", int64(2)}, int64(1))
	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE a = ? AND b = ?", []any{"y", int64(2)}, int64(1))
	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE a = ? AND b = ?", []any{"z", int64(2)}, int64(0))
}

func TestSelectWhereOrArgs(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 1)")

	assertQueryColumnArgs(t, db, "SELECT b FROM foo WHERE a = ? OR a = ?", []any{"x", "y"}, []any{int64(1), int64(2), int64(1)})
	assertQueryColumnArgs(t, db, "SELECT b FROM foo WHERE a = ? OR b = ?", []any{"x", int64(1)}, []any{int64(1), int64(2), int64(1)})
}

func TestSelectWhereArgSubExpr(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 1)")

	assertQueryColumnArgs(t, db, "SELECT b FROM foo WHERE (a = ?)", []any{"x"}, []any{int64(1), int64(2)})
	assertQueryColumnArgs(t, db, "SELECT b FROM foo WHERE (a = ? AND b = ?)", []any{"x", int64(2)}, []any{int64(2)})
}

func TestSelectWhereArgInt(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (n INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertExec(t, db, "INSERT INTO foo VALUES (7)")

	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE n = ?", []any{int64(7)}, int64(1))
	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE n > ?", []any{int64(3)}, int64(1))
	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE n <= ?", []any{int64(3)}, int64(1))
	assertQueryValueArgs(t, db, "SELECT COUNT(*) FROM foo WHERE n != ?", []any{int64(3)}, int64(1))
}

func TestSelectArgProjection(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")

	assertQueryColumnArgs(t, db, "SELECT ? FROM foo", []any{"x"}, []any{"x", "x"})
}

func TestOrderByArg(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")
	assertExec(t, db, "INSERT INTO foo VALUES ('c')")

	assertQueryColumnArgs(t, db, "SELECT label FROM foo ORDER BY label = ?", []any{"b"}, []any{"a", "c", "b"})
}

func TestGroupByWhereArg(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 3)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 4)")

	assertQueryColumnArgs(t, db, "SELECT a FROM foo WHERE b > ? GROUP BY a", []any{int64(1)}, []any{"x", "y"})
}

func TestJoinWhereArg(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 'x')")
	assertExec(t, db, "INSERT INTO foo VALUES ('2', 'y')")
	assertExec(t, db, "CREATE TABLE bar (b TEXT, c TEXT)")
	assertExec(t, db, "INSERT INTO bar VALUES ('x', '10')")
	assertExec(t, db, "INSERT INTO bar VALUES ('y', '20')")

	assertQueryRowArgs(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b WHERE bar.c = ?", []any{"20"}, []any{"2", "y", "y", "20"})
	assertQueryRowArgs(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b WHERE foo.a = ?", []any{"1"}, []any{"1", "x", "x", "10"})
}

func TestUpdateWhereArg(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")
	assertExec(t, db, "INSERT INTO foo VALUES ('d')")

	assertExec(t, db, "UPDATE foo SET label = ? WHERE label = ?", "c", "b")
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{"a", "c", "d"})
}

func TestUpdateSetLiteralWhereArg(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")

	assertExec(t, db, "UPDATE foo SET label = 'z' WHERE label = ?", "b")
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{"a", "z"})
}

func TestSelectWhereArgMissing(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")

	rows, err := db.Query("SELECT * FROM foo WHERE label = ?")
	require.NoError(t, err)
	defer rows.Close()

	require.False(t, rows.Next())
	require.Error(t, rows.Err())
	require.Contains(t, rows.Err().Error(), "missing argument")
}
