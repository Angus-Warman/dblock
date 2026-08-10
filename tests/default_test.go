package tests

import "testing"

func TestInsertWithDefault(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER, value TEXT DEFAULT 'a')")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertQueryValue(t, db, "SELECT value FROM foo", "a")
}

func TestInsertOverrideDefault(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER, value TEXT DEFAULT 'a')")
	assertExec(t, db, "INSERT INTO foo VALUES (1, 'b')")
	assertQueryValue(t, db, "SELECT value FROM foo", "b")
}

func TestInsertWithDefaultExpr(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER, b INTEGER, c INTEGER DEFAULT a + b)")
	assertExec(t, db, "INSERT INTO foo VALUES (1, 2)")
	assertQueryValue(t, db, "SELECT c FROM foo", int64(3))
}

func TestInsertDefaultValues(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER DEFAULT 42, value TEXT DEFAULT 'hello')")
	assertExec(t, db, "INSERT INTO foo DEFAULT VALUES")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{int64(42), "hello"})
}

func TestInsertNamedColumnWithDefaultKeyword(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER, first TEXT DEFAULT 'a', second TEXT DEFAULT 'b')")
	assertExec(t, db, "INSERT INTO foo VALUES (1, DEFAULT, 'c')")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{int64(1), "a", "c"})
}

func TestInsertNamedColumnOmitDefault(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER, value TEXT DEFAULT 'a')")
	assertExec(t, db, "INSERT INTO foo (id) VALUES (1)")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{int64(1), "a"})
}

func TestInsertNumericDefault(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id INTEGER, score REAL DEFAULT 0.5)")
	assertExec(t, db, "INSERT INTO foo (id) VALUES (1)")
	assertQueryValue(t, db, "SELECT score FROM foo", 0.5)
}
