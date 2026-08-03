package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectOneColumn(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT, c TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', '2', '3')")
	mustQueryOne(t, db, "SELECT b FROM foo", []any{"2"})
}

func TestSelectMultipleColumns(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT, c TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', '2', '3')")
	mustQueryOne(t, db, "SELECT c, a FROM foo", []any{"3", "1"})
	mustQueryOne(t, db, "SELECT a, b, c FROM foo", []any{"1", "2", "3"})
}

func TestSelectAllColumns(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', '2')")
	mustQueryOne(t, db, "SELECT * FROM foo", []any{"1", "2"})
}

func TestSelectUnknownColumn(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1')")
	_, err := db.Query("SELECT nope FROM foo")
	require.Error(t, err)
}

func TestSelectAliasedColumns(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b REAL)")
	rows, err := db.Query("SELECT a AS b, b AS c FROM foo")
	require.NoError(t, err)
	require.Equal(t, []string{"b", "c"}, mustColumns(t, rows))
}
