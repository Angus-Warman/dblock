package tests

import "testing"

func TestCreateIndex(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExec(t, db, "CREATE INDEX pk ON foo (id)")
	assertExec(t, db, "INSERT INTO foo VALUES ('2', 2)")
	assertQueryValue(t, db, "SELECT value FROM foo WHERE id = '2'", float64(2))
}

func TestCreateUniqueIndex(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)", "already exists")
}

func TestCreateUniqueIndexAfterInsert(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExec(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)", "already exists")
}

func TestCreateUniqueIndexAfterInserts(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExecFails(t, db, "CREATE UNIQUE INDEX pk ON foo (id)", "already exists")
}

func TestUniqueIndexAppliesToUpdate(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value TEXT)")
	assertExec(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	assertExec(t, db, "INSERT INTO foo VALUES ('aaa', 'first')")
	assertExec(t, db, "INSERT INTO foo VALUES ('bbb', 'second')")
	assertExecFails(t, db, "UPDATE foo SET id = 'aaa' WHERE id = 'bbb'", "already exists")
}
