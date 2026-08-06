package tests

import "testing"

func TestWhereAnd(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 2)")

	assertQueryColumn(t, db, "SELECT b FROM foo WHERE a = 'x' AND b = 2", []any{int64(2)})
	assertQueryColumn(t, db, "SELECT b FROM foo WHERE a = 'x' AND b = 1", []any{int64(1)})
}

func TestWhereOr(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 2)")

	assertQueryColumn(t, db, "SELECT b FROM foo WHERE a = 'x' OR b = 2", []any{int64(1), int64(2), int64(2)})
}

func TestWherePrecedence(t *testing.T) {
	// AND binds tighter than OR: a = 1 OR (a = 2 AND b = 1)
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1, 0)")
	assertExec(t, db, "INSERT INTO foo VALUES (2, 1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2, 0)")

	assertQueryColumn(t, db, "SELECT a FROM foo WHERE a = 1 OR a = 2 AND b = 1", []any{int64(1), int64(2)})
}

func TestWhereTextComparison(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('apple')")
	assertExec(t, db, "INSERT INTO foo VALUES ('banana')")
	assertExec(t, db, "INSERT INTO foo VALUES ('cherry')")

	assertQueryColumn(t, db, "SELECT * FROM foo WHERE label > 'apple'", []any{"banana", "cherry"})
	assertQueryColumn(t, db, "SELECT * FROM foo WHERE label <= 'banana'", []any{"apple", "banana"})
	assertQueryColumn(t, db, "SELECT * FROM foo WHERE label != 'banana'", []any{"apple", "cherry"})
}

func TestWhereNumericComparison(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (n INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")

	assertQueryColumn(t, db, "SELECT * FROM foo WHERE n > 1", []any{int64(2), int64(3)})
	assertQueryColumn(t, db, "SELECT * FROM foo WHERE n <= 2", []any{int64(1), int64(2)})
	assertQueryColumn(t, db, "SELECT * FROM foo WHERE n != 2", []any{int64(1), int64(3)})
}

func TestWhereArithmetic(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (n INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertExec(t, db, "INSERT INTO foo VALUES (4)")

	assertQueryColumn(t, db, "SELECT * FROM foo WHERE n * 2 > 5", []any{int64(3), int64(4)})
	assertQueryColumn(t, db, "SELECT * FROM foo WHERE n % 2 = 0", []any{int64(2), int64(4)})
}

func TestWhereNoMatch(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('apple')")
	assertQueryEmpty(t, db, "SELECT * FROM foo WHERE label = 'zzz'")
}
