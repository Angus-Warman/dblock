package tests

import (
	"database/sql"
	"testing"
)

func TestInvalidQuery(t *testing.T) {
	db := openDB(t)

	testcases := []string{
		"abcde",
		"SELECT",
		"SELECT 1",
		"SELECT FROM foo",
		"INSERT INTO foo VALUES ('b')",
	}

	for _, query := range testcases {
		t.Run(query, func(t *testing.T) {
			err := assertQueryFails(t, db, query)
			assertErrContains(t, err, "dblock: ")
		})
	}
}

func TestInvalidExec(t *testing.T) {
	db := openDB(t)

	testcases := []string{
		"abcde",
		"SELECT",
		"SELECT 1",
		"SELECT FROM foo",
		"INSERT INTO foo VALUES ('b')",
	}

	for _, query := range testcases {
		t.Run(query, func(t *testing.T) {
			err := assertExecFails(t, db, query)
			assertErrContains(t, err, "dblock: ")
		})
	}
}

func TestInvalidDSN(t *testing.T) {
	dsns := []string{
		"",
		"..",
	}

	for _, dsn := range dsns {
		t.Run("dsn: "+dsn, func(t *testing.T) {
			_, err := sql.Open("dblock", "")
			assertErrContains(t, err, "dblock: ")
		})
	}
}
