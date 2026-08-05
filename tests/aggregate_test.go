package tests

import "testing"

func TestCountAll(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertQueryValue(t, db, "SELECT COUNT(*) FROM foo", int64(3))
}

func TestCountEmpty(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertQueryValue(t, db, "SELECT COUNT(*) FROM foo", int64(0))
}

func TestCountColumnSkipsNull(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a ANY)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (NULL)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertQueryValue(t, db, "SELECT COUNT(a) FROM foo", int64(2))
}

func TestSumAvgMinMax(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertQueryValue(t, db, "SELECT SUM(a) FROM foo", int64(6))
	assertQueryValue(t, db, "SELECT AVG(a) FROM foo", 2.0)
	assertQueryValue(t, db, "SELECT MIN(a) FROM foo", int64(1))
	assertQueryValue(t, db, "SELECT MAX(a) FROM foo", int64(3))
}

func TestAggregatesSkipNull(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a ANY)")
	assertExec(t, db, "INSERT INTO foo VALUES (NULL)")
	assertExec(t, db, "INSERT INTO foo VALUES (5)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertQueryValue(t, db, "SELECT MIN(a) FROM foo", int64(2))
	assertQueryValue(t, db, "SELECT MAX(a) FROM foo", int64(5))
	assertQueryValue(t, db, "SELECT COUNT(a) FROM foo", int64(2))
}

func TestAggregatesOnEmpty(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertQueryValue(t, db, "SELECT MIN(a) FROM foo", nil)
	assertQueryValue(t, db, "SELECT MAX(a) FROM foo", nil)
	assertQueryValue(t, db, "SELECT AVG(a) FROM foo", nil)
	assertQueryValue(t, db, "SELECT SUM(a) FROM foo", 0.0)
}

func TestMinMaxText(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('banana')")
	assertExec(t, db, "INSERT INTO foo VALUES ('apple')")
	assertQueryValue(t, db, "SELECT MIN(a) FROM foo", "apple")
	assertQueryValue(t, db, "SELECT MAX(a) FROM foo", "banana")
}

func TestGroupByCount(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 3)")
	assertExec(t, db, "INSERT INTO foo VALUES ('z', 4)")
	assertExec(t, db, "INSERT INTO foo VALUES ('z', 5)")
	assertQueryRows(t, db, "SELECT a, COUNT(*) FROM foo GROUP BY a", [][]any{
		{"x", int64(2)},
		{"y", int64(1)},
		{"z", int64(2)},
	})
}

func TestGroupBySum(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 2)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 10)")
	assertQueryRows(t, db, "SELECT a, SUM(b) FROM foo GROUP BY a", [][]any{
		{"x", int64(3)},
		{"y", int64(10)},
	})
}

func TestGroupByNullGroup(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a ANY)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x')")
	assertExec(t, db, "INSERT INTO foo VALUES (NULL)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x')")
	assertExec(t, db, "INSERT INTO foo VALUES (NULL)")
	assertQueryRows(t, db, "SELECT a, COUNT(*) FROM foo GROUP BY a", [][]any{
		{"x", int64(2)},
		{nil, int64(2)},
	})
}

func TestGroupByDistinct(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x')")
	assertExec(t, db, "INSERT INTO foo VALUES ('y')")
	assertExec(t, db, "INSERT INTO foo VALUES ('x')")
	assertExec(t, db, "INSERT INTO foo VALUES ('z')")
	assertQueryColumn(t, db, "SELECT a FROM foo GROUP BY a", []any{"x", "y", "z"})
}

func TestAggregateWithWhere(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertExec(t, db, "INSERT INTO foo VALUES (2)")
	assertExec(t, db, "INSERT INTO foo VALUES (3)")
	assertQueryValue(t, db, "SELECT COUNT(*) FROM foo WHERE a > 1", int64(2))
}

func TestAggregateColumnNames(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (1)")
	assertQueryColumnNames(t, db, "SELECT COUNT(*) FROM foo", []string{"COUNT(*)"})
	assertQueryColumnNames(t, db, "SELECT COUNT(*) AS total FROM foo", []string{"total"})
}

func TestGroupByOrderBy(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES ('x', 1)")
	assertExec(t, db, "INSERT INTO foo VALUES ('z', 3)")
	assertExec(t, db, "INSERT INTO foo VALUES ('y', 2)")
	assertQueryRows(t, db, "SELECT a, COUNT(*) FROM foo GROUP BY a ORDER BY a", [][]any{
		{"x", int64(1)},
		{"y", int64(1)},
		{"z", int64(1)},
	})
}

func TestCountByExpression(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (a TEXT, b TEXT)")
	assertExec(t, db, "INSERT INTO foo VALUES ('q', 'q')")
	assertExec(t, db, "INSERT INTO foo VALUES ('q', 'q')")
	assertExec(t, db, "INSERT INTO foo VALUES ('v', 'v')")
	assertExec(t, db, "INSERT INTO foo VALUES ('v', 'v')")
	assertExec(t, db, "INSERT INTO foo VALUES ('q', 'v')")
	assertExec(t, db, "INSERT INTO foo VALUES ('v', 'q')")
	assertQueryColumn(t, db, "SELECT COUNT(*) FROM foo GROUP BY a = b", []any{int64(4), int64(2)})
}
