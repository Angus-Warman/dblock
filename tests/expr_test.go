package tests

import "testing"

func TestExprEval(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a REAL, b REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES (3.5, 4.5)")
	assertQueryValue(t, db, "SELECT a * b FROM foo", 15.75)
}

func TestExprEvalBool(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a REAL, b REAL)")
	assertExec(t, db, "INSERT INTO foo VALUES (3.5, 4.5)")
	assertQueryValue(t, db, "SELECT a + 1 = b FROM foo", true)
}

func TestExprParseBool(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a BOOL)")
	assertExec(t, db, "INSERT INTO foo VALUES (TRUE)")
	assertQueryValue(t, db, "SELECT 1 + 1 = 2 = a FROM foo", true)
}

func TestExprWhere(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertExec(t, db, "INSERT INTO foo VALUES (4)")
	assertExec(t, db, "INSERT INTO foo VALUES (5)")
	assertQueryValue(t, db, "SELECT * FROM foo WHERE a % 2 = 0", int64(4))
}
