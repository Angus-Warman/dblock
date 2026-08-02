package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataSchemaValue(t *testing.T) {
	db, fp := openFileDB(t)
	db.Exec("CREATE TABLE foo (label TEXT)")
	val := getMetadataValue(t, fp, 17)
	require.Equal(t, uint32(1), val)
}
