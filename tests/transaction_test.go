package tests

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func beginTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		tx.Rollback()
	})
	return tx
}

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

func TestReadDuringTransaction(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	tx := beginTx(t, db)
	mustExecTx(t, tx, "INSERT INTO foo VALUES ('a')")
	rows, err := tx.Query("SELECT * FROM foo")
	require.NoError(t, err)
	require.Equal(t, []any{"a"}, getColumn(t, rows, 0))
	require.NoError(t, tx.Rollback())
}

func TestCommit(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec("INSERT INTO foo VALUES ('b')")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	mustQueryColumn(t, db, "SELECT * FROM foo", 0, []any{"a", "b"})
}

func TestRollback(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx := beginTx(t, db)
	_, err := tx.Exec("INSERT INTO foo VALUES ('b')")
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	mustQueryColumn(t, db, "SELECT * FROM foo", 0, []any{"a"})
}

func TestIsolation(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx := beginTx(t, db)
	_, err := tx.Exec("INSERT INTO foo VALUES ('b')")
	require.NoError(t, err)

	// DB only sees first
	mustQueryOne(t, db, "SELECT * FROM foo", []any{"a"})

	// Tx sees both
	rows, err := tx.Query("SELECT * FROM foo")
	require.NoError(t, err)
	col := getColumn(t, rows, 0)
	require.Equal(t, []any{"a", "b"}, col)
}

// func TestReadIsolation(t *testing.T) {
// 	db := openDB(t)
// 	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
// 	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
// 	tx := beginTx(t, db)

// 	// While transaction is started, another row is added
// 	mustExec(t, db, "INSERT INTO foo VALUES ('b')")

// 	mustQueryColumn(t, db, "SELECT * FROM foo", 0, []any{"a", "b"})

// 	mustQueryColumnTx(t, tx, "SELECT * FROM foo", 0, []any{"a"})
// }
