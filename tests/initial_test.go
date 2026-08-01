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
	res, err = db.Exec("CREATE TABLE data (label TEXT, value INTEGER)")
	require.Error(t, err, "table should already exist")
}

func TestSelectTables(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	mustExec(t, db, "CREATE TABLE bar (label TEXT, value INTEGER)")
	rows, err := db.Query("SELECT * FROM dblock_schema")
	require.NoError(t, err)
	col := getColumn(t, rows, 0)
	require.Equal(t, []any{"foo", "bar"}, col)
}

func TestSelectEmpty(t *testing.T) {
	db := openDB(t)
	rows, err := db.Query("SELECT * FROM dblock_schema")
	require.NoError(t, err)
	col := getColumn(t, rows, 0)
	require.Equal(t, []any{}, col)
}

func TestInsert(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	mustExec(t, db, "INSERT INTO foo VALUES (?, ?)", "bar", 42)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{"bar", int64(42)})
}
