package tests

import "testing"

func TestExprEval(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a REAL, b REAL)")
	mustExec(t, db, "INSERT INTO foo VALUES (3.5, 4.5)")
	mustQueryOne(t, db, "SELECT a * b FROM foo", []any{15.75})
}

func TestExprEvalBool(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a REAL, b REAL)")
	mustExec(t, db, "INSERT INTO foo VALUES (3.5, 4.5)")
	mustQueryOne(t, db, "SELECT a + 1 = b FROM foo", []any{true})
}

func TestExprParseBool(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a BOOL)")
	mustExec(t, db, "INSERT INTO foo VALUES (TRUE)")
	mustQueryOne(t, db, "SELECT 1 + 1 = 2 = a FROM foo", []any{true})
}

func TestExprWhere(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (a INTEGER)")
	mustExec(t, db, "INSERT INTO foo VALUES (3)")
	mustExec(t, db, "INSERT INTO foo VALUES (4)")
	mustExec(t, db, "INSERT INTO foo VALUES (5)")
	mustQueryOne(t, db, "SELECT * FROM foo WHERE a % 2 = 0", []any{int64(4)})
}
