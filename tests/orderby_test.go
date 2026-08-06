package tests

import "testing"

func TestOrderByDesc(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('c')")

	assertQueryColumn(t, db, "SELECT a FROM foo ORDER BY a DESC", []any{"c", "b", "a"})
}

func TestOrderByMultipleColumns(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 1)")

	assertQueryColumn(t, db, "SELECT b FROM foo ORDER BY a, b", []any{int64(1), int64(2), int64(1)})
}

func TestOrderByMixedDirection(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 1)")

	assertQueryColumn(t, db, "SELECT b FROM foo ORDER BY a ASC, b DESC", []any{int64(2), int64(1), int64(1)})
}

func TestOrderByNumeric(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (n INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (30)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (20)")

	assertQueryColumn(t, db, "SELECT n FROM foo ORDER BY n", []any{int64(1), int64(20), int64(30)})
	assertQueryColumn(t, db, "SELECT n FROM foo ORDER BY n DESC", []any{int64(30), int64(20), int64(1)})
}

func TestOrderByTieStable(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 'first')")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 'second')")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 'third')")

	assertQueryColumn(t, db, "SELECT b FROM foo ORDER BY a", []any{"first", "second", "third"})
}
