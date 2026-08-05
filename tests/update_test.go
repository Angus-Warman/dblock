package tests

import "testing"

func TestUpdate(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertQueryValue(t, db, "SELECT * FROM foo", int64(1))
	assertExec(t, db, "UPDATE foo SET v = 2")
	assertQueryValue(t, db, "SELECT * FROM foo", int64(2))
}

func TestUpdateWhere(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertExec(t, db, "INSERT INTO foo VALUES ('b')")
	assertExec(t, db, "INSERT INTO foo VALUES ('d')")
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{"a", "b", "d"})
	assertExec(t, db, "UPDATE foo SET v = 'c' WHERE v = 'd'")
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{"a", "b", "c"})
}

func TestUpdateByExpression(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertQueryValue(t, db, "SELECT * FROM foo", int64(1))
	assertExec(t, db, "UPDATE foo SET v = v + 1")
	assertQueryValue(t, db, "SELECT * FROM foo", int64(2))
}
