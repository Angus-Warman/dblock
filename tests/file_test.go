package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveToFile(t *testing.T) {
	db, fp := openFileDB(t)
	require.NotNil(t, db)
	mustQueryColumn(t, db, "SELECT * FROM dblock_schema", 0, []any{})
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	mustExec(t, db, "INSERT INTO foo VALUES (?, ?)", "bar", 42)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{"bar", int64(42)})
	require.NoError(t, db.Close())

	// Reopen the same file: schema and data must survive.
	reopened := reopenFileDB(t, fp)
	require.NotNil(t, reopened)
	mustQueryColumn(t, reopened, "SELECT * FROM dblock_schema", 0, []any{"foo"})
	mustQueryOne(t, reopened, "SELECT * FROM foo", []any{"bar", int64(42)})

	// A new table must not reuse a page that already exists on disk.
	mustExec(t, reopened, "CREATE TABLE bar (label TEXT)")
	mustExec(t, reopened, "INSERT INTO bar VALUES (?)", "baz")
	mustQueryOne(t, reopened, "SELECT * FROM bar", []any{"baz"})
}

func TestMetadata(t *testing.T) {
	db, fp := openFileDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	db.Close()

	expectedMetadata := []byte("dblock")
	padding := make([]byte, 100-6)
	expectedMetadata = append(expectedMetadata, padding...)
	buf, err := os.ReadFile(fp)
	require.NoError(t, err)
	require.Equal(t, expectedMetadata, buf[0:100])
}
