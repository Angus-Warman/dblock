package tests

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

type Execer interface {
	Exec(string, ...any) (sql.Result, error)
}

type Queryer interface {
	Query(string, ...any) (*sql.Rows, error)
}

func assertExec(t *testing.T, db Execer, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err)
}

func assertExecFails(t *testing.T, db Execer, query string, errMessage string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.Error(t, err)
	require.Contains(t, err.Error(), errMessage)
}

func assertQueryFails(t *testing.T, db Queryer, query string, errMessage string, args ...any) {
	t.Helper()
	_, err := db.Query(query, args...)
	require.Error(t, err)
	require.Contains(t, err.Error(), errMessage)
}

func assertQueryRows(t *testing.T, db *sql.DB, query string, expectedValues [][]any) {
	t.Helper()
	rows, err := db.Query(query)
	require.NoError(t, err)
	values := scanRows(t, rows)
	require.Equal(t, expectedValues, values)
}

func assertQueryRow(t *testing.T, db Queryer, query string, expectedRow []any) {
	t.Helper()
	sqlRows, err := db.Query(query)
	require.NoError(t, err)
	rows := scanRows(t, sqlRows)
	require.Len(t, rows, 1, "query should return exactly one row")
	require.Equal(t, expectedRow, rows[0])
}

func assertQueryValue(t *testing.T, db Queryer, query string, expectedValue any) {
	t.Helper()
	sqlRows, err := db.Query(query)
	require.NoError(t, err)
	rows := scanRows(t, sqlRows)
	require.Len(t, rows, 1, "query should return exactly one row")
	row := rows[0]
	require.Len(t, row, 1, "query should return a row with exactly one value")
	require.Equal(t, expectedValue, row[0])
}

func assertQueryValueArgs(t *testing.T, db Queryer, query string, args []any, expectedValue any) {
	t.Helper()
	sqlRows, err := db.Query(query, args...)
	require.NoError(t, err)
	rows := scanRows(t, sqlRows)
	require.Len(t, rows, 1, "query should return exactly one row")
	row := rows[0]
	require.Len(t, row, 1, "query should return a row with exactly one value")
	require.Equal(t, expectedValue, row[0])
}

func assertQueryRowArgs(t *testing.T, db Queryer, query string, args []any, expectedRow []any) {
	t.Helper()
	sqlRows, err := db.Query(query, args...)
	require.NoError(t, err)
	rows := scanRows(t, sqlRows)
	require.Len(t, rows, 1, "query should return exactly one row")
	require.Equal(t, expectedRow, rows[0])
}

func assertQueryRowsArgs(t *testing.T, db *sql.DB, query string, args []any, expectedValues [][]any) {
	t.Helper()
	sqlRows, err := db.Query(query, args...)
	require.NoError(t, err)
	values := scanRows(t, sqlRows)
	require.Equal(t, expectedValues, values)
}

func assertQueryColumnArgs(t *testing.T, db Queryer, query string, args []any, expectedCol []any) {
	t.Helper()
	sqlRows, err := db.Query(query, args...)
	require.NoError(t, err)
	cols, err := sqlRows.Columns()
	require.NoError(t, err)
	require.Len(t, cols, 1, "query should return exactly one column")
	rows := scanRows(t, sqlRows)
	actualColumn := []any{}

	for _, row := range rows {
		actualColumn = append(actualColumn, row[0])
	}

	require.Equal(t, expectedCol, actualColumn)
}

func assertQueryEmpty(t *testing.T, db Queryer, query string) {
	t.Helper()
	sqlRows, err := db.Query(query)
	require.NoError(t, err)
	rows := scanRows(t, sqlRows)
	require.Len(t, rows, 0, "expected exactly 0 rows")
}

func assertQueryColumn(t *testing.T, db Queryer, query string, expectedCol []any) {
	t.Helper()
	sqlRows, err := db.Query(query)
	require.NoError(t, err)
	cols, err := sqlRows.Columns()
	require.NoError(t, err)
	require.Len(t, cols, 1, "query should return exactly one column")
	rows := scanRows(t, sqlRows)
	actualColumn := []any{}

	for _, row := range rows {
		actualColumn = append(actualColumn, row[0])
	}

	require.Equal(t, expectedCol, actualColumn)
}

func assertQueryColumnNames(t *testing.T, db Queryer, query string, expectedNames []string) {
	t.Helper()
	sqlRows, err := db.Query(query)
	require.NoError(t, err)
	actualNames, err := sqlRows.Columns()
	require.NoError(t, err)
	require.Equal(t, expectedNames, actualNames)
}

func scanRows(t *testing.T, rows *sql.Rows) [][]any {
	t.Helper()

	colNames, err := rows.Columns()
	require.NoError(t, err)
	rowValues := [][]any{}

	defer rows.Close()

	for rows.Next() {
		require.NoError(t, rows.Err())
		rowVals := make([]any, len(colNames))
		rowPtrs := make([]any, len(colNames))

		for i := range rowVals {
			rowPtrs[i] = &rowVals[i]
		}

		err := rows.Scan(rowPtrs...)
		require.NoError(t, err)
		rowValues = append(rowValues, rowVals)
	}

	return rowValues
}
