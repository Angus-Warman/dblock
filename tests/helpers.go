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
