package internal

import (
	"dblock2/internal/parser"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type ExprKind uint8

const (
	StarExpr ExprKind = iota + 1
	ColumnExpr
	IntExpr
	RealExpr
	BlobExpr
	TextExpr
	// BoolExpr
	BinaryKind
	FuncKind
)

type Expr struct {
	Kind     ExprKind
	Star     bool
	Column   string
	Int      int64
	Real     float64
	Blob     []byte
	Text     string
	Bool     bool
	Binary   *BinaryExpr
	FuncCall *FuncExpr
}

type FuncName string

const (
	CountFunc FuncName = "COUNT"
	MaxFunc   FuncName = "MAX"
	MinFunc   FuncName = "MIN"
	SumFunc   FuncName = "SUM"
	AvgFunc   FuncName = "AVG"
)

type FuncExpr struct {
	Name FuncName
	Args []Expr
}

type Operator string

const (
	// arithmetic
	Add      Operator = "+"
	Subtract Operator = "-"
	Multiply Operator = "*"
	Divide   Operator = "/"
	Modulo   Operator = "%"

	// comparison
	Equal           Operator = "="
	NotEqual        Operator = "!="
	LessThan        Operator = "<"
	LessThanOrEq    Operator = "<="
	GreaterThan     Operator = ">"
	GreaterThanOrEq Operator = ">="

	// logical
	And Operator = "AND"
	Or  Operator = "OR"
)

type BinaryExpr struct {
	Left  *Expr
	Op    Operator
	Right *Expr
}

var precedence = map[Operator]int{
	Or:    1,
	And:   2,
	Equal: 3, NotEqual: 3, LessThan: 3, LessThanOrEq: 3, GreaterThan: 3, GreaterThanOrEq: 3,
	Add: 4, Subtract: 4,
	Multiply: 5, Divide: 5, Modulo: 5,
}

var cmpAliases = map[string]Operator{
	"+":   Add,
	"-":   Subtract,
	"*":   Multiply,
	"/":   Divide,
	"%":   Modulo,
	"=":   Equal,
	"==":  Equal,
	"!=":  NotEqual,
	"<>":  NotEqual,
	"<":   LessThan,
	"<=":  LessThanOrEq,
	">":   GreaterThan,
	">=":  GreaterThanOrEq,
	"AND": And,
	"OR":  Or,
}

// resolveExpr is the entrypoint: it takes the raw, flat parser.Expr (a
// sequence of Factors as produced by the grammar) and resolves it into a
// proper AST honoring operator precedence and associativity.
func resolveExpr(input *parser.Expr) (*Expr, error) {
	if input == nil {
		return nil, fmt.Errorf("resolveExpr: nil expr")
	}
	if len(input.Factors) == 0 {
		return nil, fmt.Errorf("resolveExpr: empty expr")
	}

	m := &exprMachine{factors: input.Factors}
	expr, err := m.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if !m.done() {
		return nil, fmt.Errorf("resolveExpr: unexpected token at %d, %+v", m.pos, m.factors[m.pos])
	}
	return expr, nil
}

// exprMachine walks a flat []parser.Factor and, via precedence climbing,
// resolves it into a tree of Expr/BinaryExpr nodes. It's a small state
// machine over a cursor position rather than a recursive grammar rule,
// which keeps precedence-handling separate from parsing.
type exprMachine struct {
	factors []parser.Factor
	pos     int
}

func (m *exprMachine) done() bool {
	return m.pos >= len(m.factors)
}

func (m *exprMachine) peek() *parser.Factor {
	if m.done() {
		return nil
	}
	return &m.factors[m.pos]
}

func (m *exprMachine) next() (parser.Factor, error) {
	if m.done() {
		return parser.Factor{}, fmt.Errorf("resolveExpr: unexpected end of expression")
	}
	f := m.factors[m.pos]
	m.pos++
	return f, nil
}

// operatorOf extracts the operator token (if any) from a Factor, normalized
// via cmpAliases. Returns ok=false if the factor is not an operator token.
func operatorOf(f *parser.Factor) (Operator, bool) {
	if f.Star { // Resolves this ambiguity
		return Multiply, true
	}

	if f.Op == "" {
		return "", false
	}

	op, ok := cmpAliases[f.Op]
	return op, ok
}

// parseExpr implements precedence climbing: it consumes a primary value,
// then greedily folds in any following binary operators whose precedence
// is >= minPrec, recursing with minPrec+1 on the right-hand side to enforce
// left-associativity.
func (m *exprMachine) parseExpr(minPrec int) (*Expr, error) {
	left, err := m.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		f := m.peek()
		if f == nil {
			break
		}
		op, ok := operatorOf(f)
		if !ok {
			break
		}
		prec, ok := precedence[op]
		if !ok || prec < minPrec {
			break
		}
		m.pos++ // consume operator

		right, err := m.parseExpr(prec + 1)
		if err != nil {
			return nil, err
		}

		left = &Expr{
			Kind: BinaryKind,
			Binary: &BinaryExpr{
				Left:  left,
				Op:    op,
				Right: right,
			},
		}
	}

	return left, nil
}

// parsePrimary resolves a single non-operator Factor: a literal, column
// reference, star, function call, or parenthesized sub-expression.
func (m *exprMachine) parsePrimary() (*Expr, error) {
	f, err := m.next()
	if err != nil {
		return nil, err
	}

	switch {
	case f.Star:
		return &Expr{Kind: StarExpr, Star: true}, nil

	case f.Func != nil:
		return resolveFuncCall(f.Func)

	case f.SubExpr != nil:
		sub := &exprMachine{factors: f.SubExpr.Factors}
		expr, err := sub.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if !sub.done() {
			return nil, fmt.Errorf("resolveExpr: unexpected trailing token in parenthesized expression")
		}
		return expr, nil

	case f.Column != "":
		return &Expr{Kind: ColumnExpr, Column: f.Column}, nil

	case f.Num != "":
		return parseNumLiteral(f.Num)

	case f.Hex != "":
		blob, err := parseHexLiteral(f.Hex)
		if err != nil {
			return nil, err
		}
		return &Expr{Kind: BlobExpr, Blob: blob}, nil

	case f.Str != "":
		return &Expr{Kind: TextExpr, Text: unquoteString(f.Str)}, nil

	default:
		return nil, fmt.Errorf("resolveExpr: unrecognized factor at position %d", m.pos-1)
	}
}

func resolveFuncCall(fc *parser.FuncCall) (*Expr, error) {
	name := FuncName(fc.Name)

	switch name {
	case CountFunc, MinFunc, MaxFunc, SumFunc, AvgFunc:
		// do nothing
	default:
		return nil, fmt.Errorf("%q function not supported", name)
	}

	args := make([]Expr, len(fc.Args))
	for i, argExpr := range fc.Args {
		sub := &exprMachine{factors: argExpr.Factors}
		resolved, err := sub.parseExpr(0)
		if err != nil {
			return nil, fmt.Errorf("resolveExpr: arg %d of %s: %w", i, fc.Name, err)
		}
		if !sub.done() {
			return nil, fmt.Errorf("resolveExpr: unexpected trailing token in arg %d of %s", i, fc.Name)
		}
		args[i] = *resolved
	}
	return &Expr{
		Kind: FuncKind,
		FuncCall: &FuncExpr{
			Name: name,
			Args: args,
		},
	}, nil
}

// parseBoolLiteral recognizes bare TRUE/FALSE keywords that the lexer
// tokenizes as an identifier (Column), since the grammar has no dedicated
// boolean token. Case-insensitive per SQL convention.
func parseBoolLiteral(s string) (bool, bool) {
	switch strings.ToUpper(s) {
	case "TRUE":
		return true, true
	case "FALSE":
		return false, true
	default:
		return false, false
	}
}

// parseNumLiteral decides between IntExpr and RealExpr based on whether the
// literal text looks like a float (contains '.', 'e', or 'E').
func parseNumLiteral(s string) (*Expr, error) {
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("resolveExpr: invalid number literal %q: %w", s, err)
		}
		return &Expr{Kind: RealExpr, Real: f}, nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("resolveExpr: invalid number literal %q: %w", s, err)
	}
	return &Expr{Kind: IntExpr, Int: i}, nil
}

// parseHexLiteral decodes a hex/blob literal (e.g. SQLite's x'53514C697465')
// into raw bytes, stripping any x'...' / X'...' wrapper if present.
func parseHexLiteral(s string) ([]byte, error) {
	trimmed := s
	if len(trimmed) >= 3 {
		upper := strings.ToUpper(trimmed)
		if (strings.HasPrefix(upper, "X'") || strings.HasPrefix(upper, "0X")) && strings.HasSuffix(trimmed, "'") {
			trimmed = strings.TrimSuffix(trimmed[2:], "'")
		} else if strings.HasPrefix(upper, "0X") {
			trimmed = trimmed[2:]
		}
	}
	b, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("resolveExpr: invalid hex literal %q: %w", s, err)
	}
	return b, nil
}
