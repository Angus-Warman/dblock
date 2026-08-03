package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTimeRoundtrip(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (created TIME)")
	now := time.Now().In(time.UTC)
	mustExec(t, db, "INSERT INTO foo VALUES (?)", now)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{now})
}

func TestUUIDRoundtrip(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (id UUID)")
	original := uuid.New()
	mustExec(t, db, "INSERT INTO foo VALUES (?)", original)
	var returned uuid.UUID
	rows, err := db.Query("SELECT * FROM foo")
	require.NoError(t, err)
	rows.Next()
	err = rows.Scan(&returned)
	require.NoError(t, err)
	require.Equal(t, original, returned)
}
