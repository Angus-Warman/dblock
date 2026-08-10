package parser

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
)

func Parse(query string) (*ParsedStmt, error) {
	s, err := sqlParser.ParseString("", query)

	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return s, nil
}

var sqlExprParser = participle.MustBuild[Expr](
	participle.Lexer(sqlLexer),
)

func ParseExpr(query string) (*Expr, error) {
	e, err := sqlExprParser.ParseString("", query)

	if err != nil {
		return nil, fmt.Errorf("parse expression: %v: %w", query, err)
	}

	return e, nil
}

// RenderExpr serializes a parsed expression back to SQL text.
func RenderExpr(e *Expr) string {
	var sb strings.Builder

	for _, f := range e.Factors {
		sb.WriteString(renderFactor(f))
	}

	return sb.String()
}

func renderFactor(f Factor) string {
	switch {
	case f.Star:
		return "*"

	case f.Func != nil:
		args := make([]string, len(f.Func.Args))

		for i := range args {
			args[i] = RenderExpr(&f.Func.Args[i])
		}

		return f.Func.Name + "(" + strings.Join(args, ", ") + ")"

	case f.SubExpr != nil:
		return "(" + RenderExpr(f.SubExpr) + ")"

	case f.Num != "":
		return f.Num

	case f.Hex != "":
		return f.Hex

	case f.Str != "":
		return f.Str

	case f.Op != "":
		return " " + f.Op + " "

	case f.Arg != "":
		return "?"

	case f.Column != "":
		return f.Column
	}

	return ""
}
