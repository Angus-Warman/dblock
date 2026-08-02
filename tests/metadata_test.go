package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataSchemaChange(t *testing.T) {
	db, fp := openFileDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db, "CREATE TABLE bar (label TEXT)")
	require.NoError(t, db.Close())
	val := getMetadataValue(t, fp, 17) // Schema change
	require.Equal(t, uint32(2), val)
}

func TestMetadataFileChange(t *testing.T) {
	db, fp := openFileDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('a')")
	require.NoError(t, db.Close())
	val := getMetadataValue(t, fp, 13) // File change
	require.Equal(t, uint32(1), val)
}
