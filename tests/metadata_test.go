package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataSchemaValue(t *testing.T) {
	db, fp := openFileDB(t)
	mustExec(t, db, "CREATE TABLE foo (label TEXT)")
	require.NoError(t, db.Close())
	val := getMetadataValue(t, fp, 17)
	require.Equal(t, uint32(1), val)
}
