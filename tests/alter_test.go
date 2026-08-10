package tests

import "testing"

func TestAlterColumn(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES ('a')")
	assertErrContains(t, err, "expects INTEGER")
	assertExec(t, db, "ALTER TABLE foo ALTER COLUMN v TYPE ANY")
	assertExec(t, db, "INSERT INTO foo VALUES ('a')")
	assertQueryColumn(t, db, "SELECT * FROM foo", []any{int64(1), "a"})
	err = assertExecFails(t, db, "ALTER TABLE foo ALTER COLUMN v TYPE TEXT")
	assertErrContains(t, err, "data loss")
}

func TestAlterRenameTable(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (7)")
	assertExec(t, db, "ALTER TABLE foo RENAME TO bar")
	err := assertQueryFails(t, db, "SELECT * FROM foo")
	assertErrContains(t, err, "'foo' does not exist")
	assertQueryColumn(t, db, "SELECT * FROM bar", []any{int64(7)})
	assertQueryColumn(t, db, "SELECT name FROM dblock_schema", []any{"bar"})
}

func TestAlterRenameTableCollision(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "CREATE TABLE bar (v INTEGER)")
	err := assertExecFails(t, db, "ALTER TABLE foo RENAME TO bar")
	assertErrContains(t, err, "'bar' already exists")
	assertQueryColumn(t, db, "SELECT name FROM dblock_schema", []any{"foo", "bar"})
}

func TestAlterRenameColumn(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (7)")
	assertExec(t, db, "ALTER TABLE foo RENAME COLUMN v TO w")
	assertQueryColumn(t, db, "SELECT w FROM foo", []any{int64(7)})
	err := assertQueryFails(t, db, "SELECT v FROM foo")
	assertErrContains(t, err, "'v' does not exist")
	assertExec(t, db, "INSERT INTO foo VALUES (8)")
	assertQueryColumn(t, db, "SELECT w FROM foo", []any{int64(7), int64(8)})
}

func TestAlterRenameColumnMissing(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (v INTEGER)")
	err := assertExecFails(t, db, "ALTER TABLE foo RENAME COLUMN nope TO w")
	assertErrContains(t, err, "no such column")
}

func TestAlterTableAddColumn(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (x INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "ALTER TABLE foo ADD COLUMN y TEXT")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{int64(1), ""})
}

func TestAlterTableAddColumnWithDefault(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (x INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "ALTER TABLE foo ADD COLUMN y INTEGER DEFAULT 2")
	assertQueryRow(t, db, "SELECT * FROM foo", []any{int64(1), int64(2)})
}
