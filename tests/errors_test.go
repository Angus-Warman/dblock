package tests

import "testing"

func TestGarbageIn(t *testing.T) {
	db := openDB(t)

	type testcase struct {
		query  string
		errMsg string
	}

	testcases := []testcase{
		{
			query:  "abcde",
			errMsg: "",
		},
		{
			query:  "SELECT",
			errMsg: "",
		},
		{
			query:  "SELECT 1",
			errMsg: "",
		},
		{
			query:  "SELECT FROM foo",
			errMsg: "",
		},
		// {
		// 	query:  "INSERT INTO foo VALUES ('b')",
		// 	errMsg: "TODO",
		// },
	}

	for _, tc := range testcases {
		t.Run(tc.query, func(t *testing.T) {
			assertQueryFails(t, db, tc.query, tc.errMsg)
		})
	}
}
