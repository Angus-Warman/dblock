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
	joins      []JoinStmt
	orders     []ColumnRef
	where      *Expr
}

type ProjectedColumn struct {
	source string
	alias  string
	expr   *Expr
}

type JoinMode string

const (
	BareJoin       JoinMode = "" // resolves to INNER
	InnerJoin      JoinMode = "INNER"
	LeftJoin       JoinMode = "LEFT" // resolves to RIGHT OUTER
	LeftOuterJoin  JoinMode = "LEFT OUTER"
	RightJoin      JoinMode = "RIGHT" // resolves to RIGHT OUTER
	RightOuterJoin JoinMode = "RIGHT OUTER"
	FullJoin       JoinMode = "FULL" // resolves to FULL OUTER
	FullOuterJoin  JoinMode = "FULL OUTER"
	CrossJoin      JoinMode = "CROSS" /// not currently implemented
)

type JoinStmt struct {
	tableName string
	onExpr    *Expr
	mode      JoinMode
}

type ColumnRef struct {
	table  string
	column string
}
