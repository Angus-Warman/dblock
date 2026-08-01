package tests

import (
	"database/sql"
	_ "dblock2"
	"testing"

	"github.com/stretchr/testify/require"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("dblock", ":memory:")
	require.NoError(t, err)
	return db
}

func TestPing(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.Ping())
}

func TestPrepare(t *testing.T) {
	db := openDB(t)
	prep, err := db.Prepare("CREATE TABLE data (label TEXT, value INTEGER)")
	require.NoError(t, err)
	require.NotNil(t, prep)
}

func TestExecute(t *testing.T) {
	db := openDB(t)
	res, err := db.Exec("CREATE TABLE data (label TEXT, value INTEGER)")
	require.NoError(t, err)
	require.NotNil(t, res)
	res, err = db.Exec("CREATE TABLE data (label TEXT, value INTEGER)")
	require.Error(t, err, "table should already exist")
}

func TestSelectTables(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	rows, err := db.Query("SELECT * FROM dblock_schema")
	require.NoError(t, err)
	col := getColumn(t, rows, 0)
	require.Equal(t, []any{"foo"}, col)
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	_, err := db.Exec(query)
	require.NoError(t, err)
}

func getColumn(t *testing.T, rows *sql.Rows, colIdx int) []any {
	t.Helper()

	colNames, err := rows.Columns()
	require.NoError(t, err)
	colValues := []any{}

	for rows.Next() {
		require.NoError(t, rows.Err())
		rowVals := make([]any, len(colNames))
		rowPtrs := make([]any, len(colNames))

		for i := range rowVals {
			rowPtrs[i] = &rowVals[i]
		}

		err := rows.Scan(rowPtrs...)
		require.NoError(t, err)
		colValues = append(colValues, rowVals[colIdx])
	}

	return colValues
}
