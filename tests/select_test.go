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

func TestJoinTables(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', '2')")
	mustExec(t, db, "CREATE TABLE bar (b TEXT, c TEXT)")
	mustExec(t, db, "INSERT INTO bar VALUES ('2', '3')")
	mustQueryOne(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b", []any{"1", "2", "3"})
	mustQueryColumnNames(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b", []string{"foo.a", "b", "bar.c"})
}

func TestJoinMultipleRows(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', 'x')")
	mustExec(t, db, "INSERT INTO foo VALUES ('2', 'y')")
	mustExec(t, db, "INSERT INTO foo VALUES ('3', 'z')")
	mustExec(t, db, "CREATE TABLE bar (b TEXT, c TEXT)")
	mustExec(t, db, "INSERT INTO bar VALUES ('y', '10')")
	mustExec(t, db, "INSERT INTO bar VALUES ('w', '20')")
	rows, err := db.Query("SELECT * FROM foo JOIN bar ON foo.b = bar.b")
	require.NoError(t, err)
	require.Equal(t, [][]any{{"2", "y", "10"}}, getValues(t, rows))
}

func TestSelectOrderBy(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('2', 'x')")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', 'y')")
	mustExec(t, db, "INSERT INTO foo VALUES ('3', 'z')")

	mustRows(t, db, "SELECT * FROM foo ORDER BY a", [][]any{{"1", "y"}, {"2", "x"}, {"3", "z"}})
}
