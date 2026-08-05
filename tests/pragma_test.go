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

func TestPragmaFileRoundtrip(t *testing.T) {
	db1, fp := openFileDB(t)
	assertExec(t, db1, "PRAGMA token -1234")
	require.NoError(t, db1.Close())
	db2 := reopenFileDB(t, fp)
	assertQueryValue(t, db2, "PRAGMA token", int64(-1234))
}
