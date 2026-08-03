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

func TestCatchInvalidData(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value REAL)")
	_, err := db.Exec("INSERT INTO foo VALUES (?)", "bar")
	require.Error(t, err)
}

func TestCatchIntegerMismatch(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value INTEGER)")
	_, err := db.Exec("INSERT INTO foo VALUES (?)", "bar")
	require.Error(t, err)
}

func TestCatchTextMismatch(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value TEXT)")
	_, err := db.Exec("INSERT INTO foo VALUES (?)", 42)
	require.Error(t, err)
}

func TestInsertNull(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value INTEGER)")
	_, err := db.Exec("INSERT INTO foo VALUES (?)", nil)
	require.Error(t, err)
}

func TestInsertNullIntoAny(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value ANY)")
	mustExec(t, db, "INSERT INTO foo VALUES (?)", nil)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{nil})
}

func TestSemiRoundtripInt(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (value INTEGER)")
	original := int32(1234)
	mustExec(t, db, "INSERT INTO foo VALUES (?)", original)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{int64(1234)})
}
