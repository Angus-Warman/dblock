package tests

import (
	"database/sql"
	"dblock2/internal/metadata"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("dblock", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err)
}

func mustExecTx(t *testing.T, tx *sql.Tx, query string, args ...any) {
	t.Helper()
	_, err := tx.Exec(query, args...)
	require.NoError(t, err)
}

func mustQueryOne(t *testing.T, db *sql.DB, query string, expectedRow []any) {
	t.Helper()
	rows, err := db.Query(query)
	require.NoError(t, err)
	rowVals := getRows(t, rows)
	require.Len(t, rowVals, 1)
	require.Equal(t, expectedRow, rowVals[0])
}

func mustQueryColumn(t *testing.T, db *sql.DB, query string, colIdx int, expectedCol []any) {
	t.Helper()
	rows, err := db.Query(query)
	require.NoError(t, err)
	colVals := getColumn(t, rows, colIdx)
	require.Equal(t, expectedCol, colVals)
}

func mustQueryColumnTx(t *testing.T, tx *sql.Tx, query string, colIdx int, expectedCol []any) {
	t.Helper()
	rows, err := tx.Query(query)
	require.NoError(t, err)
	colVals := getColumn(t, rows, colIdx)
	require.Equal(t, expectedCol, colVals)
}

func getColumn(t *testing.T, rows *sql.Rows, colIdx int) []any {
	t.Helper()

	rowValues := getRows(t, rows)

	colValues := []any{}

	for rowIdx := range rowValues {
		require.Less(t, colIdx, len(rowValues[rowIdx]))
		colValues = append(colValues, rowValues[rowIdx][colIdx])
	}

	return colValues
}

func getRows(t *testing.T, rows *sql.Rows) [][]any {
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

func beginTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	require.NoError(t, err)
	t.Cleanup(func() {
		tx.Rollback()
	})
	return tx
}

func openFileDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	fp := filepath.Join(t.TempDir(), t.Name())
	db, err := sql.Open("dblock", fp)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db, fp
}

func reopenFileDB(t *testing.T, fp string) *sql.DB {
	t.Helper()
	db, err := sql.Open("dblock", fp)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}

func getMetadata(t *testing.T, fp string) *metadata.Metadata {
	t.Helper()
	f, err := os.Open(fp)
	require.NoError(t, err)
	defer f.Close()
	buf := make([]byte, 100)
	f.ReadAt(buf, 0)
	m, err := metadata.Decode(buf)
	require.NoError(t, err)
	return m
}
