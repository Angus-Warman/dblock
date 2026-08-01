package internal

type ExecStmt struct {
	createStmt *CreateStmt
	insertStmt *InsertStmt
}

type CreateStmt struct {
	tableName   string
	columnNames []string
}

type InsertStmt struct {
	tableName string
	values    []any // Either a placholder ?, or parsed literal value
}

type QueryStmt struct {
	tableName string
}
