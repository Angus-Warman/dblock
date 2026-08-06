package tests

import (
	"database/sql"
	"os"
	"testing"

	"dblock2/internal/metadata"

	"github.com/stretchr/testify/require"
)

func TestSaveToFile(t *testing.T) {
	db, fp := openFileDB(t)
	require.NotNil(t, db)
	assertQueryEmpty(t, db, "SELECT * FROM dblock_schema")
	assertExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	assertExec(t, db, "INSERT INTO foo VALUES (?, ?)", "bar", 42)
	assertQueryRow(t, db, "SELECT * FROM foo", []any{"bar", int64(42)})
	require.NoError(t, db.Close())

	// Reopen the same file: schema and data must survive.
	reopened := reopenFileDB(t, fp)
	require.NotNil(t, reopened)
	assertQueryColumn(t, reopened, "SELECT name FROM dblock_schema", []any{"foo"})
	assertQueryRow(t, reopened, "SELECT * FROM foo", []any{"bar", int64(42)})

	// A new table must not reuse a page that already exists on disk.
	assertExec(t, reopened, "CREATE TABLE bar (label TEXT)")
	assertExec(t, reopened, "INSERT INTO bar VALUES (?)", "baz")
	assertQueryValue(t, reopened, "SELECT * FROM bar", "baz")
}

func TestMetadata(t *testing.T) {
	db, fp := openFileDB(t)
	assertExec(t, db, "CREATE TABLE foo (label TEXT, value INTEGER)")
	db.Close()

	buf, err := os.ReadFile(fp)
	require.NoError(t, err)

	m, err := metadata.Decode(buf[:metadata.Length])
	require.NoError(t, err)
	require.Equal(t, "dblock01", m.Dblock)
	require.Equal(t, uint8(13), m.PageSizePower)
	require.Equal(t, uint32(2), m.NumberOfPages)
	require.Equal(t, uint32(2), m.FileVersion)
	require.Equal(t, uint32(2), m.SchemaVersion)
	require.Equal(t, int64(0), m.Token)

	// The file must start with exactly the encoded metadata and no more.
	require.Equal(t, m.Encode(), buf[:metadata.Length])
}

func TestOpenFileTwice(t *testing.T) {
	_, fp := openFileDB(t)

	// Without closing, open second db. Should fail.
	_, err := sql.Open("dblock", fp)
	require.Error(t, err)
}

func TestSetPageSize(t *testing.T) {
	db, fp := openFileDB(t)
	assertExec(t, db, "PRAGMA page_size 8")
	assertExec(t, db, "CREATE TABLE foo (a TEXT)")
	require.NoError(t, db.Close())
	stat, err := os.Stat(fp)
	require.NoError(t, err)
	require.Equal(t, int64(256), stat.Size())
}
