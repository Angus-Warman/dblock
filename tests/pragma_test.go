package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPragmaRoundtrip(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "PRAGMA token -1234")
	assertQueryValue(t, db, "PRAGMA token", int64(-1234))
}

func TestReadMetadata(t *testing.T) {
	db := openDB(t)
	rows, err := db.Query("PRAGMA read_metadata")
	require.NoError(t, err)
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.Equal(t, []string{"property", "value"}, cols)

	props := map[string]any{}
	values := scanRows(t, rows)

	for _, row := range values {
		props[row[0].(string)] = row[1]
	}

	require.Equal(t, "dblock01", props["dblock"])
	require.Equal(t, int64(1), props["file_version"])
	require.Equal(t, int64(1), props["schema_version"])
	require.Equal(t, int64(13), props["page_size_power"])
	require.Equal(t, int64(0), props["number_of_pages"])
	require.Equal(t, int64(0), props["token"])
	require.Len(t, props, 7)
}

func TestPragmaFileRoundtrip(t *testing.T) {
	db1, fp := openFileDB(t)
	assertExec(t, db1, "PRAGMA token -1234")
	require.NoError(t, db1.Close())
	db2 := reopenFileDB(t, fp)
	assertQueryValue(t, db2, "PRAGMA token", int64(-1234))
}
