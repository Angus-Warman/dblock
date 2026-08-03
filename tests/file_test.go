package tests

import (
	"os"
	"testing"

	"dblock2/internal/metadata"

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

	buf, err := os.ReadFile(fp)
	require.NoError(t, err)

	m, err := metadata.Decode(buf[:metadata.Length])
	require.NoError(t, err)
	require.Equal(t, "dblock", m.Dblock)
	require.Equal(t, uint16(1), m.DatabaseVersion)
	require.Equal(t, uint8(13), m.PageSizePower)
	require.Equal(t, uint32(0), m.NumberOfPages)
	require.Equal(t, uint32(1), m.FileChangeCounter)
	require.Equal(t, uint32(1), m.SchemaChangeCounter)
	require.Equal(t, uint32(0), m.TokenValue)

	// The file must start with exactly the encoded metadata and no more.
	require.Equal(t, m.Encode(), buf[:metadata.Length])
}
