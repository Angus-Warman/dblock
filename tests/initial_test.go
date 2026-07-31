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
}
