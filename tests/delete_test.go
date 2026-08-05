package tests

import "testing"

func TestDropTable(t *testing.T) {
	db := openDB(t)
	assertExecFails(t, db, "DROP TABLE foo", "'foo' does not exist")
	assertExec(t, db, "CREATE TABLE foo (v ANY)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertQueryValue(t, db, "SELECT name FROM dblock_schema", "foo")
	assertExec(t, db, "DROP TABLE foo")
	assertQueryEmpty(t, db, "SELECT name FROM dblock_schema")
	assertQueryFails(t, db, "SELECT * FROM foo", "'foo' does not exist")
	assertExec(t, db, "DROP TABLE IF EXISTS bar")

	// Create table with same name
	assertExec(t, db, "CREATE TABLE foo (v ANY)")
	assertQueryEmpty(t, db, "SELECT * FROM foo")
}
