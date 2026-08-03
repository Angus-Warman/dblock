package internal

import "dblock2/internal/pragma"

type ExecStmt struct {
	createStmt *CreateStmt
	insertStmt *InsertStmt
	pragmaStmt *PragmaStmt
}

type CreateStmt struct {
	tableName string
	columns   []Column
}

type InsertStmt struct {
	tableName string
	values    []any // Either a placholder ?, or parsed literal value
}

type PragmaStmt struct {
	property pragma.Property
	value    string
}

type QueryStmt struct {
	selectStmt *SelectStmt
	pragmaStmt *PragmaStmt
}

type SelectStmt struct {
	tableName  string
	projection []ProjectedColumn
}

type ProjectedColumn struct {
	source string
	alias  string
}
