package tests

import (
	"testing"
)

func TestSharedInMemory(t *testing.T) {
	db1 := openDB(t)
	mustExec(t, db1, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db1, "CREATE TABLE bar (label TEXT)")
	db2 := openDB(t)
	mustQueryColumn(t, db2, "SELECT * FROM dblock_schema", 0, []any{"foo", "bar"})
}

func TestSharedInMemoryAfterClose(t *testing.T) {
	db1 := openDB(t)
	mustExec(t, db1, "CREATE TABLE foo (label TEXT)")
	db1.Close()
	db2 := openDB(t)
	mustQueryColumn(t, db2, "SELECT * FROM dblock_schema", 0, []any{})
}

// func TestIsolation(t *testing.T) {
// 	db := openDB(t)
// 	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
// 	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
// 	tx, err := db.Begin()
// 	require.NoError(t, err)
// 	_, err = tx.Exec("INSERT INTO foo VALUES ('b')")
// 	require.NoError(t, err)

// 	// DB only sees first
// 	mustQueryOne(t, db, "SELECT * FROM foo", []any{"a"})

// 	// Tx sees both
// 	rows, err := tx.Query("SELECT * FROM foo")
// 	require.NoError(t, err)
// 	col := getColumn(t, rows, 0)
// 	require.Equal(t, []any{"a", "b"}, col)
// }

// func TestCommit(t *testing.T) {
// 	db := openDB(t)
// 	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
// 	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
// 	tx, err := db.Begin()
// 	require.NoError(t, err)
// 	_, err = tx.Exec("INSERT INTO foo VALUES ('b')")
// 	require.NoError(t, err)
// 	require.NoError(t, tx.Commit())
// 	mustQueryOne(t, db, "SELECT * FROM foo", []any{"a", "b"})
// }

// func TestRollback(t *testing.T) {
// 	db := openDB(t)
// 	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
// 	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
// 	tx, err := db.Begin()
// 	require.NoError(t, err)
// 	_, err = tx.Exec("INSERT INTO foo VALUES ('b')")
// 	require.NoError(t, err)
// 	require.NoError(t, tx.Rollback())
// 	mustQueryOne(t, db, "SELECT * FROM foo", []any{"a"})
// }
