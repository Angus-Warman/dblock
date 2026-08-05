package tests

import (
	"fmt"
	"strings"
	"testing"
)

func TestSelectOneColumn(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT, c TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', '2', '3')")
	assertQueryValue(t, db, "SELECT b FROM foo", "2")
}

func TestSelectMultipleColumns(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT, c TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', '2', '3')")
	assertQueryRow(t, db, "SELECT c, a FROM foo", []any{"3", "1"})
	assertQueryRow(t, db, "SELECT a, b, c FROM foo", []any{"1", "2", "3"})
}

func TestSelectAllColumns(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', '2')")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{"1", "2"})
}

func TestSelectUnknownColumn(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1')")
	assertQueryFails(t, db, "SELECT nope FROM foo", `no such column "nope"`)
}

func TestSelectAliasedColumns(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b REAL)")
	assertQueryColumnNames(t, db, "SELECT a AS b, b AS c FROM foo", []string{"b", "c"})
}

func TestJoinTables(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', '2')")
	assertExec(t, db, "CREATE TABLE bar (b TEXT, c TEXT)")
	assertExec(t, db, "INSERT INTO bar VALUES ('2', '3')")
	assertQueryRow(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b", []any{"1", "2", "2", "3"})
	assertQueryColumnNames(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b", []string{"foo.a", "foo.b", "bar.b", "bar.c"})
}

func TestJoinMultipleRows(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 'x')")
	assertExec(t, db, "INSERT INTO foo VALUES ('2', 'y')")
	assertExec(t, db, "INSERT INTO foo VALUES ('3', 'z')")
	assertExec(t, db, "CREATE TABLE bar (b TEXT, c TEXT)")
	assertExec(t, db, "INSERT INTO bar VALUES ('y', '10')")
	assertExec(t, db, "INSERT INTO bar VALUES ('w', '20')")
	assertQueryRows(t, db, "SELECT * FROM foo JOIN bar ON foo.b = bar.b", [][]any{{"2", "y", "y", "10"}})
}

func TestSelectOrderBy(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('2', 'x')")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 'y')")
	assertExec(t, db, "INSERT INTO foo VALUES ('3', 'z')")

	assertQueryRows(t, db, "SELECT * FROM foo ORDER BY a", [][]any{{"1", "y"}, {"2", "x"}, {"3", "z"}})
}

func TestSelectMany(t *testing.T) {
	db := openDB(t)

	numTables := 10
	numCols := 10
	numRows := 10

	for tIdx := 0; tIdx < numTables; tIdx++ {
		colDefs := make([]string, numCols)

		for cIdx := range numCols {
			colDefs[cIdx] = fmt.Sprintf("c%d TEXT", cIdx)
		}

		assertExec(t, db, fmt.Sprintf("CREATE TABLE t%d (%s)", tIdx, strings.Join(colDefs, ", ")))

		values := make([]string, numCols)

		for cIdx := range numCols {
			values[cIdx] = fmt.Sprintf("'v%d'", cIdx)
		}

		for range numRows {
			assertExec(t, db, fmt.Sprintf("INSERT INTO t%d VALUES (%s)", tIdx, strings.Join(values, ", ")))
		}
	}

	for tIdx := range numTables {
		expectedRow := make([]any, numCols)

		for cIdx := range numCols {
			expectedRow[cIdx] = fmt.Sprintf("v%d", cIdx)
		}

		expectedRows := make([][]any, numRows)

		for rIdx := range numRows {
			expectedRows[rIdx] = expectedRow
		}

		assertQueryRows(t, db, fmt.Sprintf("SELECT * FROM t%d", tIdx), expectedRows)
	}
}
