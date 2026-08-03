package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPragmaRoundtrip(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "PRAGMA token -1234")
	mustQueryColumn(t, db, "PRAGMA token", 0, []any{int64(-1234)})
}

func TestPragmaFileRoundtrip(t *testing.T) {
	db1, fp := openFileDB(t)
	mustExec(t, db1, "PRAGMA token -1234")
	require.NoError(t, db1.Close())
	db2 := reopenFileDB(t, fp)
	mustQueryColumn(t, db2, "PRAGMA token", 0, []any{int64(-1234)})
}
