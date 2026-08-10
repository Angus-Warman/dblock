package parser

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var sqlLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "ws", Pattern: `\s+`},
	{Name: "Keyword", Pattern: `CREATE|DROP|TABLE|INDEX|PRIMARY|KEY|UNIQUE|DEFAULT|INSERT|INTO|VALUES|SELECT|FROM|WHERE|IF|NOT|EXISTS|UPDATE|SET|GROUP|ORDER|ASC|DESC|AS|LIMIT|OFFSET|JOIN|INNER|OUTER|LEFT|RIGHT|FULL|CROSS|ON|ALTER|RENAME|COLUMN|TO`},
	{Name: "TypeName", Pattern: `TEXT|INTEGER|REAL|BLOB|BOOL|TIME|UUID|ANY`},
	{Name: "True", Pattern: `TRUE`},
	{Name: "False", Pattern: `FALSE`},
	{Name: "Null", Pattern: `NULL`},
	{Name: "Uuid", Pattern: `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`},
	{Name: "Logic", Pattern: `AND|OR`},
	{Name: "Ident", Pattern: `[a-zA-Z_][\w]*(?:\.[a-zA-Z_][\w]*)*`},
	{Name: "String", Pattern: `'[^']*'|"[^"]*"|` + "`[^`]*`"},
	{Name: "Hex", Pattern: `0x[0-9a-fA-F]+`},
	{Name: "Number", Pattern: `-?\d+(?:\.\d+)?`},
	{Name: "Arg", Pattern: `\?`},
	{Name: "Punct", Pattern: `[(),;\.]`},
	{Name: "Op", Pattern: `[*+\-%/]`},
	{Name: "Cmp", Pattern: `>=|<=|<>|!=|>|<|=`},
})

var sqlParser = participle.MustBuild[ParsedStmt](
	participle.Lexer(sqlLexer),
)

type ParsedStmt struct {
	Alter     *AlterStmt     `parser:"  @@  \";\"?"`
	Create    *CreateStmt    `parser:"| @@  \";\"?"`
	Insert    *InsertStmt    `parser:"| @@  \";\"?"`
	Select    *SelectStmt    `parser:"| @@  \";\"?"`
	Update    *UpdateStmt    `parser:"| @@  \";\"?"`
	Pragma    *PragmaStmt    `parser:"| @@  \";\"?"`
	Drop      *DropStmt      `parser:"| @@  \";\"?"`
	CreateIdx *CreateIdxStmt `parser:"| @@  \";\"?"`
}

type AlterStmt struct {
	IfExists  *IfExistsClause `parser:"\"ALTER\" \"TABLE\" (@@)?"`
	Name      string          `parser:"@Ident"`
	RenameCol *RenameColOp    `parser:"( \"RENAME\" \"COLUMN\" @@ )?"`
	RenameTbl *RenameTblOp    `parser:"( \"RENAME\" \"TO\" @@ )?"`
	AlterCol  *AlterColTypeOp `parser:"( \"ALTER\" \"COLUMN\" @@ )?"`
	AddCol    *AddColumnOp    `parser:"( \"ADD\" \"COLUMN\" @@ )?"`
}

func (a *AlterStmt) HasIfExists() bool { return a.IfExists != nil }

type RenameColOp struct {
	OldName string `parser:"@Ident"`
	NewName string `parser:"\"TO\" @Ident"`
}

type RenameTblOp struct {
	NewName string `parser:"@Ident"`
}

type AlterColTypeOp struct {
	ColName string `parser:"@Ident"`
	NewType string `parser:"\"TYPE\" @TypeName"`
}

type AddColumnOp struct {
	Name    string         `parser:"@Ident"`
	Type    string         `parser:"@TypeName"`
	Default *DefaultClause `parser:"(@@)?"`
}

type CreateStmt struct {
	ExistClause *IfNotExistsClause `parser:"\"CREATE\" \"TABLE\" (@@)?"`
	TableName   string             `parser:"@Ident"`
	Columns     []ParsedColumn     `parser:"\"(\" @@ (\",\" @@)* \")\""`
}

type CreateIdxStmt struct {
	IsUnique    bool               `parser:"\"CREATE\" @(\"UNIQUE\")?"`
	ExistClause *IfNotExistsClause `parser:"\"INDEX\" (@@)?"`
	IndexName   string             `parser:"@Ident"`
	TableName   string             `parser:"\"ON\" @Ident"`
	Columns     []string           `parser:"\"(\" @Ident (\",\" @Ident)* \")\""`
}

type InsertStmt struct {
	TableName string      `parser:"\"INSERT\" \"INTO\" @Ident"`
	Columns   []string    `parser:"(\"(\" @Ident (\",\" @Ident)* \")\")?"`
	Body      *InsertBody `parser:"@@"`
}

type InsertBody struct {
	Default *DefaultValuesClause `parser:"  @@"`
	Values  *InsertValuesClause  `parser:"| \"VALUES\" \"(\" @@ (\",\" @@)* \")\""`
}

type DefaultValuesClause struct {
	Line string `parser:"\"DEFAULT\" \"VALUES\""`
}

type InsertValuesClause struct {
	Values []Value `parser:"@@ (\",\" @@)*"`
}

type SelectStmt struct {
	List      SelectList     `parser:"\"SELECT\" @@"`
	TableName string         `parser:"(\"FROM\" @Ident)?"`
	Alias     string         `parser:"((\"AS\")? @Ident)?"`
	Joins     []JoinClause   `parser:"(@@)*"`
	Where     *WhereClause   `parser:"(@@)?"`
	GroupBy   *GroupByClause `parser:"(@@)?"`
	OrderBy   *OrderByClause `parser:"(@@)?"`
	Limit     *LimitClause   `parser:"(@@)?"`
	Offset    *OffsetClause  `parser:"(@@)?"`
}

type JoinClause struct {
	Mode  string  `parser:"(@(\"INNER\"|\"LEFT\"|\"RIGHT\"|\"FULL\"|\"CROSS\") \"OUTER\"?)?"`
	Table string  `parser:"\"JOIN\" @Ident"`
	Alias string  `parser:"((\"AS\")? @Ident)?"`
	On    *JoinOn `parser:"@@"`
}

type JoinOn struct {
	Expr Expr `parser:"\"ON\" @@"`
}

type ColumnAlias string

func (c ColumnAlias) String() string {
	return string(c)
}

func (c ColumnAlias) HasPrefix() bool {
	return strings.Contains(string(c), ".")
}

func (c ColumnAlias) Table() string {
	s := string(c)
	if before, _, ok := strings.Cut(s, "."); ok {
		return before
	}
	return ""
}

func (c ColumnAlias) Column() string {
	s := string(c)
	if _, after, ok := strings.Cut(s, "."); ok {
		return after
	}
	return s
}

type UpdateStmt struct {
	Name   string       `parser:"\"UPDATE\" @Ident"`
	Column string       `parser:"\"SET\" @Ident"`
	Value  Expr         `parser:"\"=\" @@"`
	Where  *WhereClause `parser:"(@@)?"`
}

type IfNotExistsClause struct {
	Line string `parser:"\"IF\" \"NOT\" \"EXISTS\""`
}

type IfExistsClause struct {
	Line string `parser:"\"IF\" \"EXISTS\""`
}

type ParsedColumn struct {
	Name         string         `parser:"@Ident"`
	Type         string         `parser:"@TypeName"`
	IsPrimaryKey bool           `parser:"@(\"PRIMARY\" \"KEY\")?"`
	IsUnique     bool           `parser:"@(\"UNIQUE\")?"`
	Default      *DefaultClause `parser:"(@@)?"`
}

type DefaultClause struct {
	Expr Expr `parser:"\"DEFAULT\" @@"`
}

type WhereClause struct {
	Expr Expr `parser:"\"WHERE\" @@"`
}

type GroupByClause struct {
	Items []Expr `parser:"\"GROUP\" \"BY\" @@ (\",\" @@)*"`
}

type OrderByClause struct {
	Items []OrderItem `parser:"\"ORDER\" \"BY\" @@ (\",\" @@)*"`
}

type OrderItem struct {
	Expr Expr   `parser:"@@"`
	Dir  string `parser:"(@(\"ASC\" | \"DESC\"))?"`
}

type LimitClause struct {
	Count string `parser:"\"LIMIT\" @Number"`
}

type OffsetClause struct {
	Count string `parser:"\"OFFSET\" @Number"`
}

type Value struct {
	Num     string `parser:"  @Number"`
	Bytes   string `parser:"| @Hex"`
	Str     string `parser:"| @String"`
	Null    string `parser:"| @Null"`
	Bool    string `parser:"| @True | @False"`
	Uuid    string `parser:"| @Uuid"`
	Arg     string `parser:"| @Arg"`
	Default string `parser:"| @\"DEFAULT\""`
}

type SelectItem struct {
	Star  bool   `parser:"  @\"*\""`
	Expr  *Expr  `parser:"| @@"`
	Alias string `parser:"(\"AS\" @Ident)?"`
}

type Expr struct {
	Factors []Factor `parser:"(@@)*"`
}

type Factor struct {
	Star    bool      `parser:"  @\"*\""`
	Func    *FuncCall `parser:"| @@"`
	SubExpr *Expr     `parser:"| \"(\" @@ \")\""`
	Num     string    `parser:"| @Number"`
	Hex     string    `parser:"| @Hex"`
	Str     string    `parser:"| @String"`
	Op      string    `parser:"| @Op | @Cmp | @Logic"`
	Arg     string    `parser:"| @Arg"`
	Column  string    `parser:"| @Ident"`
}

type FuncCall struct {
	Name string `parser:"@Ident \"(\""`
	Args []Expr `parser:"(@@ (\",\" @@)*)? \")\""`
}

type SelectList struct {
	Items []SelectItem `parser:"@@ (\",\" @@)*"`
}

type PragmaStmt struct {
	Property string `parser:"\"PRAGMA\" @Ident"`
	Value    string `parser:"(@(Ident|Number))?"`
}

type DropStmt struct {
	ExistClause *IfExistsClause `parser:"\"DROP\" \"TABLE\" (@@)?"`
	TableName   string          `parser:"@Ident"`
}
