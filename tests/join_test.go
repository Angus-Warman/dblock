package tests

import "testing"

func TestJoinOnExpression(t *testing.T) {
	db := openDB(t)

	mustExec(t, db, "CREATE TABLE foo (a INTEGER, b INTEGER)")
	mustExec(t, db, "INSERT INTO foo VALUES (1, 10)")
	mustExec(t, db, "INSERT INTO foo VALUES (5, 50)")

	mustExec(t, db, "CREATE TABLE bar (c INTEGER)")
	mustExec(t, db, "INSERT INTO bar VALUES (2)")
	mustExec(t, db, "INSERT INTO bar VALUES (6)")

	mustRows(t, db, "SELECT * FROM foo JOIN bar ON foo.a + 1 = bar.c", [][]any{
		{int64(1), int64(10), int64(2)},
		{int64(5), int64(50), int64(6)},
	})
}

func TestJoinModes(t *testing.T) {
	db := openDB(t)

	mustExec(t, db, "CREATE TABLE foo (id ANY, a TEXT)")
	mustExec(t, db, "INSERT INTO foo VALUES ('1', 'w')")
	mustExec(t, db, "INSERT INTO foo VALUES ('2', 'x')")
	mustExec(t, db, "INSERT INTO foo VALUES ('3', 'y')")
	mustExec(t, db, "INSERT INTO foo VALUES ('4', 'z')")
	mustExec(t, db, "INSERT INTO foo VALUES ('4', 'z2')") // fan-out on foo side too
	mustExec(t, db, "INSERT INTO foo VALUES (NULL, 'n')") // NULL key

	mustExec(t, db, "CREATE TABLE bar (id ANY, b TEXT)")
	mustExec(t, db, "INSERT INTO bar VALUES ('3', 'i')")
	mustExec(t, db, "INSERT INTO bar VALUES ('4', 'j')")
	mustExec(t, db, "INSERT INTO bar VALUES ('5', 'k')")
	mustExec(t, db, "INSERT INTO bar VALUES ('5', 'l')")
	mustExec(t, db, "INSERT INTO bar VALUES (NULL, 'n2')") // NULL key

	type testcase struct {
		query string
		rows  [][]any
	}

	testcases := []testcase{
		{
			query: "SELECT * FROM foo",
			rows: [][]any{
				{"1", "w"},
				{"2", "x"},
				{"3", "y"},
				{"4", "z"},
				{"4", "z2"},
				{nil, "n"},
			},
		},
		{
			query: "SELECT * FROM foo JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
			},
		},
		{
			query: "SELECT * FROM foo INNER JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
			},
		},
		{
			// LEFT [OUTER] JOIN: every foo row kept, unmatched bar side is NULL.
			// NULL = NULL never matches, so foo's NULL row stays unmatched.
			query: "SELECT * FROM foo LEFT JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"1", "w", nil, nil},
				{"2", "x", nil, nil},
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, "n", nil, nil},
			},
		},
		{
			query: "SELECT * FROM foo LEFT OUTER JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"1", "w", nil, nil},
				{"2", "x", nil, nil},
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, "n", nil, nil},
			},
		},
		{
			query: "SELECT * FROM foo RIGHT JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, nil, "5", "k"},
				{nil, nil, "5", "l"},
				{nil, nil, nil, "n2"},
			},
		},
		{
			query: "SELECT * FROM foo RIGHT OUTER JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, nil, "5", "k"},
				{nil, nil, "5", "l"},
				{nil, nil, nil, "n2"},
			},
		},
		{
			// symmetric to above with tables swapped
			query: "SELECT * FROM bar LEFT JOIN foo ON bar.id = foo.id",
			rows: [][]any{
				{"3", "i", "3", "y"},
				{"4", "j", "4", "z"},
				{"4", "j", "4", "z2"},
				{"5", "k", nil, nil},
				{"5", "l", nil, nil},
				{nil, "n2", nil, nil},
			},
		},
		{
			query: "SELECT * FROM bar RIGHT JOIN foo ON bar.id = foo.id",
			rows: [][]any{
				{"3", "i", "3", "y"},
				{"4", "j", "4", "z"},
				{"4", "j", "4", "z2"},
				{nil, nil, "1", "w"},
				{nil, nil, "2", "x"},
				{nil, nil, nil, "n"},
			},
		},
		{
			query: "SELECT * FROM foo FULL JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"1", "w", nil, nil},
				{"2", "x", nil, nil},
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, "n", nil, nil},
				{nil, nil, "5", "k"},
				{nil, nil, "5", "l"},
				{nil, nil, nil, "n2"},
			},
		},
		{
			query: "SELECT * FROM foo FULL OUTER JOIN bar ON foo.id = bar.id",
			rows: [][]any{
				{"1", "w", nil, nil},
				{"2", "x", nil, nil},
				{"3", "y", "3", "i"},
				{"4", "z", "4", "j"},
				{"4", "z2", "4", "j"},
				{nil, "n", nil, nil},
				{nil, nil, "5", "k"},
				{nil, nil, "5", "l"},
				{nil, nil, nil, "n2"},
			},
		},
		// TODO(next): CROSS JOIN without ON clause (optional On in grammar + cross-product mode in JoinScanner).
		// {
		// 	query: "SELECT COUNT(*) FROM foo CROSS JOIN bar",
		// 	rows: [][]any{
		// 		{int64(30)},
		// 	},
		// },
		// TODO(next): comma-join `FROM foo, bar` as implicit join + WHERE clause execution.
		// {
		// 	// join predicate in WHERE clause == implicit inner join
		// 	query: "SELECT * FROM foo, bar WHERE foo.id = bar.id",
		// 	rows: [][]any{
		// 		{"3", "y", "3", "i"},
		// 		{"4", "z", "4", "j"},
		// 		{"4", "z2", "4", "j"},
		// 	},
		// },
		// TODO(next): AND conditions in the ON predicate.
		// {
		// 	// self join
		// 	query: "SELECT f1.id, f1.a, f2.a FROM foo f1 JOIN foo f2 ON f1.id = f2.id AND f1.a != f2.a",
		// 	rows: [][]any{
		// 		{"4", "z", "z2"},
		// 		{"4", "z2", "z"},
		// 	},
		// },
	}

	for _, tc := range testcases {
		t.Run(tc.query, func(t *testing.T) {
			mustRows(t, db, tc.query, tc.rows)
		})
	}
}
