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
	err := assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertErrContains(t, err, "already exists")
}

func TestCreateUniqueIndexAfterInsert(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExec(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertErrContains(t, err, "already exists")
}

func TestCreateUniqueIndexAfterInserts(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	err := assertExecFails(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	assertErrContains(t, err, "already exists")
}

func TestUniqueIndexAppliesToUpdate(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT, value TEXT)")
	assertExec(t, db, "CREATE UNIQUE INDEX pk ON foo (id)")
	assertExec(t, db, "INSERT INTO foo VALUES ('aaa', 'first')")
	assertExec(t, db, "INSERT INTO foo VALUES ('bbb', 'second')")
	err := assertExecFails(t, db, "UPDATE foo SET id = 'aaa' WHERE id = 'bbb'")
	assertErrContains(t, err, "already exists")
}

func TestUniqueColumnCreatesUniqueIndex(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT UNIQUE, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertErrContains(t, err, "already exists")
	assertQueryValue(t, db, "SELECT value FROM foo WHERE id = '1'", float64(1))
}

func TestPrimaryKeyColumnCreatesUniqueIndex(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT PRIMARY KEY, value REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('1', 1)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES ('1', 1)")
	assertErrContains(t, err, "already exists")
	assertQueryValue(t, db, "SELECT value FROM foo WHERE id = '1'", float64(1))
}

func TestPrimaryKeyEnforcedOnUpdate(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id TEXT PRIMARY KEY, value TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('aaa', 'first')")
	assertExec(t, db, "INSERT INTO foo VALUES ('bbb', 'second')")
	err := assertExecFails(t, db, "UPDATE foo SET id = 'aaa' WHERE id = 'bbb'")
	assertErrContains(t, err, "already exists")
}

func TestMultiplePrimaryKeysIsError(t *testing.T) {
	db := openDB(t)
	err := assertExecFails(t, db, "CREATE TABLE foo (id TEXT PRIMARY KEY, value TEXT PRIMARY KEY)")
	assertErrContains(t, err, "multiple primary keys")
}
