package tests

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func openFileDB(t *testing.T, fp string) *sql.DB {
	t.Helper()
	db, err := sql.Open("dblock", fp)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	return db
}

func TestSaveToFile(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "data.db")

	db := openFileDB(t, fp)
	require.NotNil(t, db)
	mustQueryColumn(t, db, "SELECT * FROM dblock_schema", 0, []any{})
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	mustExec(t, db, "INSERT INTO foo VALUES (?, ?)", "bar", 42)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{"bar", int64(42)})
	require.NoError(t, db.Close())

	// Reopen the same file: schema and data must survive.
	reopened := openFileDB(t, fp)
	require.NotNil(t, reopened)
	mustQueryColumn(t, reopened, "SELECT * FROM dblock_schema", 0, []any{"foo"})
	mustQueryOne(t, reopened, "SELECT * FROM foo", []any{"bar", int64(42)})

	// A new table must not reuse a page that already exists on disk.
	mustExec(t, reopened, "CREATE TABLE bar (label TEXT)")
	mustExec(t, reopened, "INSERT INTO bar VALUES (?)", "baz")
	mustQueryOne(t, reopened, "SELECT * FROM bar", []any{"baz"})
}
