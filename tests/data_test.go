package tests

import (
	"fmt"
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

func TestRoundtripUUIDThroughAny(t *testing.T) {
	db := openDB(t)
	mustExec(t, db, "CREATE TABLE foo (id ANY)")
	original := uuid.New()
	mustExec(t, db, "INSERT INTO foo VALUES (?)", original)
	mustQueryOne(t, db, "SELECT * FROM foo", []any{original})
}

func TestDataCoercion(t *testing.T) {
	type testCase struct {
		colType string
		in      any
		out     any
	}

	testCases := []testCase{
		{colType: "INTEGER", in: 1, out: int64(1)},
		{colType: "TEXT", in: 1, out: "1"},
		{colType: "REAL", in: 2, out: 2.0},
		{colType: "BLOB", in: "bar", out: []byte("bar")},
		{colType: "TEXT", in: []byte("bar"), out: "bar"},
		{colType: "TEXT", in: time.Unix(0, 0).UTC(), out: "1970-01-01T00:00:00Z"},
		{colType: "BOOL", in: 1, out: true},
		{colType: "BOOL", in: "TRUE", out: true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			db := openDB(t)
			tableDef := fmt.Sprintf("CREATE TABLE foo (value %v)", tc.colType)
			mustExec(t, db, tableDef)
			mustExec(t, db, "INSERT INTO foo VALUES (?)", tc.in)
			mustQueryOne(t, db, "SELECT * FROM foo", []any{tc.out})
		})
	}
}
