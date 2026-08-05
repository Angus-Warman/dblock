package tests

import (
	_ "dblock2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.Ping())
}

func TestPrepare(t *testing.T) {
	db := openDB(t)
	prep, err := db.Prepare("CREATE TABLE data (label TEXT, value INTEGER)")
	require.NoError(t, err)
	require.NotNil(t, prep)
}

func TestExecute(t *testing.T) {
	db := openDB(t)
	res, err := db.Exec("CREATE TABLE data (label TEXT, value INTEGER)")
	require.NoError(t, err)
	require.NotNil(t, res)
	assertExecFails(t, db, "CREATE TABLE data (label TEXT, value INTEGER)", "already exists")
}

func TestSelectTables(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	assertExec(t, db, "CREATE TABLE bar (label TEXT, value INTEGER)")
	assertQueryColumn(t, db, "SELECT name FROM dblock_schema", []any{"foo", "bar"})
}

func TestSelectEmpty(t *testing.T) {
	db := openDB(t)
	assertQueryEmpty(t, db, "SELECT * FROM dblock_schema")
}

func TestInsert(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (?, ?)", "bar", 42)
	assertQueryRow(t, db, "SELECT * FROM foo", []any{"bar", int64(42)})
}

func TestMultipleTables(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE a (label TEXT)")
	assertExec(t, db, "CREATE TABLE b (label TEXT)")
	assertExec(t, db, "INSERT INTO a VALUES (?)", "a value")
	assertExec(t, db, "INSERT INTO b VALUES (?)", "b value")
	assertQueryValue(t, db, "SELECT * FROM a", "a value")
	assertQueryValue(t, db, "SELECT * FROM b", "b value")
}

func TestInsertLiteral(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('bar')")
	assertQueryValue(t, db, "SELECT * FROM foo", "bar")
}

func TestInsertMixedLiteral(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a', ?)", "b")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{"a", "b"})
}
