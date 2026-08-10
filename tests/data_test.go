package tests

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTimeRoundtrip(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (created TIME)")
	now := time.Now().In(time.UTC)
	assertExec(t, db, "INSERT INTO foo VALUES (?)", now)
	assertQueryValue(t, db, "SELECT * FROM foo", now)
}

func TestUUIDRoundtripA(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id UUID)")
	original := uuid.New()
	assertExec(t, db, "INSERT INTO foo VALUES (?)", original)
	mustReturnUUID(t, db, "SELECT * FROM foo", original)
}

// This test doesn't currently pass, sql.DB cannot materialise a uuid into an *any
// func TestUUIDRoundtripB(t *testing.T) {
// 	db := openDB(t)
// 	mustExec(t, db, "CREATE TABLE foo (id UUID)")
// 	original := uuid.New()
// 	mustExec(t, db, "INSERT INTO foo VALUES (?)", original)
// 	mustQueryOne(t, db, "SELECT * FROM foo", []any{original})
// }

func TestCatchInvalidData(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (value REAL)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES (?)", "bar")
	assertErrContains(t, err, "expects REAL, got string")
}

func TestCatchIntegerMismatch(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (value INTEGER)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES (?)", 1.5)
	assertErrContains(t, err, "expects INTEGER, got float")
}

func TestInsertNull(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (value INTEGER)")
	err := assertExecFails(t, db, "INSERT INTO foo VALUES (?)", nil)
	assertErrContains(t, err, "cannot insert null value")
}

func TestInsertNullIntoAny(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (value ANY)")
	assertExec(t, db, "INSERT INTO foo VALUES (?)", nil)
	assertQueryValue(t, db, "SELECT * FROM foo", nil)
}

func TestRoundtripUUIDThroughAny(t *testing.T) {
	db := openDB(t)
	assertExec(t, db, "CREATE TABLE foo (id ANY)")
	original := uuid.New()
	assertExec(t, db, "INSERT INTO foo VALUES (?)", original)
	assertQueryValue(t, db, "SELECT * FROM foo", original)
}

func mustReturnUUID(t *testing.T, db *sql.DB, query string, original uuid.UUID) {
	t.Helper()
	var returned uuid.UUID
	rows, err := db.Query(query)
	require.NoError(t, err)
	rows.Next()
	err = rows.Scan(&returned)
	require.NoError(t, err)
	require.Equal(t, original, returned)
}

func TestInsertUUIDVariants(t *testing.T) {
	original := uuid.New()
	variants := []any{
		original,
		original.String(),
		original[:],
	}

	for _, variant := range variants {
		t.Run(fmt.Sprint(variant), func(t *testing.T) {
			db := openDB(t)
			assertExec(t, db, "CREATE TABLE foo (id UUID)")
			assertExec(t, db, "INSERT INTO foo VALUES (?)", variant)
			mustReturnUUID(t, db, "SELECT * FROM foo", original)
		})
	}
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
			assertExec(t, db, tableDef)
			assertExec(t, db, "INSERT INTO foo VALUES (?)", tc.in)
			assertQueryValue(t, db, "SELECT * FROM foo", tc.out)
		})
	}
}
