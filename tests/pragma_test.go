package tests

import "testing"

func TestPragmaRoundtrip(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "PRAGMA token 1234")
	mustQueryColumn(t, db, "PRAGMA token", 0, []any{int64(1234)})
}
