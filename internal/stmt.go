package internal

import "dblock2/internal/pragma"

type ExecStmt struct {
	createStmt *CreateStmt
	insertStmt *InsertStmt
	dropStmt   *DropStmt
	pragmaStmt *PragmaStmt
	updateStmt *UpdateStmt
	alterStmt  *AlterStmt
}

type CreateStmt struct {
	tableName string
	columns   []Column
}

type DropStmt struct {
	ifExists  bool
	tableName string
}

type InsertStmt struct {
	tableName string
	values    []any // Either a placholder ?, or parsed literal value
}

type UpdateStmt struct {
	tableName string
	column    string
	expr      *Expr
	where     *Expr
}

type AlterStmt struct {
	ifExists  bool
	tableName string
	renameCol *RenameColumnOp
	renameTbl *RenameTableOp
	alterCol  *AlterColumnTypeOp
}

type RenameColumnOp struct {
	oldName string
	newName string
}

type RenameTableOp struct {
	newName string
}

type AlterColumnTypeOp struct {
	colName string
	newType string
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
	orders     []OrderStmt
	where      *Expr
	groupBy    []*Expr
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
