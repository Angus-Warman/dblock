package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedInMemoryAfterClose(t *testing.T) {
	db1 := openDB(t)
	assertExec(t, db1, "CREATE TABLE foo (label TEXT)")
	db1.Close()
	db2 := openDB(t)
	assertQueryEmpty(t, db2, "SELECT * FROM dblock_schema")
}

func TestReadDuringTransaction(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	tx := beginTx(t, db)
	assertExec(t, tx, "INSERT INTO foo VALUES ('a')")
	assertQueryValue(t, tx, "SELECT * FROM foo", "a")
	require.NoError(t, tx.Rollback())
}

func TestCommit(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx, err := db.Begin()
	require.NoError(t, err)
	assertExec(t, tx, "INSERT INTO foo VALUES ('b')")
	require.NoError(t, tx.Commit())
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{"a", "b"})
}

func TestRollback(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx := beginTx(t, db)
	assertExec(t, tx, "INSERT INTO foo VALUES ('b')")
	require.NoError(t, tx.Rollback())
	assertQueryValue(t, db, "SELECT * FROM foo", "a")
}

func TestIsolation(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx := beginTx(t, db)
	assertExec(t, tx, "INSERT INTO foo VALUES ('b')")

	// DB only sees first
	assertQueryValue(t, db, "SELECT * FROM foo", "a")

	// Tx sees both
	assertQueryColumn(t, tx, "SELECT * FROM foo", []any{"a", "b"})
}

func TestReadIsolation(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	tx1 := beginTx(t, db)
	tx2 := beginTx(t, db)

	assertExec(t, tx1, "INSERT INTO foo VALUES ('b')")
	assertQueryColumn(t, tx1, "SELECT * FROM foo", []any{"a", "b"})
	assertQueryValue(t, tx2, "SELECT * FROM foo", "a")
}
