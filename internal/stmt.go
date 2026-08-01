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
}

type QueryStmt struct {
	tableName string
}
