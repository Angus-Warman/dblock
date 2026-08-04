package internal

import (
	"dblock2/internal/parser"
	"dblock2/internal/pragma"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func resolveSelect(parsed *parser.SelectStmt) (*QueryStmt, error) {
	if parsed.TableName == "" {
		return nil, fmt.Errorf("table name empty")
	}

	projection, err := resolveProjection(parsed.Select.Items)

	if err != nil {
		return nil, err
	}

	joins, err := resolveJoins(parsed.Joins)

	if err != nil {
		return nil, err
	}

	orders, err := resolveOrders(parsed.OrderBy, parsed.TableName, parsed.Alias)

	if err != nil {
		return nil, err
	}

	sel := &SelectStmt{
		tableName:  parsed.TableName,
		projection: projection,
		joins:      joins,
		orders:     orders,
	}

	queryStmt := &QueryStmt{
		selectStmt: sel,
	}

	return queryStmt, nil
}

func resolveJoins(parsed []parser.JoinClause) ([]JoinStmt, error) {
	joins := []JoinStmt{}

	for _, j := range parsed {
		if j.On == nil {
			return nil, fmt.Errorf("join without ON clause")
		}

		joins = append(joins, JoinStmt{
			tableName: j.Table,
			on: JoinOn{
				left: ColumnRef{
					table:  j.On.Left.Table(),
					column: j.On.Left.Column(),
				},
				right: ColumnRef{
					table:  j.On.Right.Table(),
					column: j.On.Right.Column(),
				},
			},
		})
	}

	return joins, nil
}

func resolveOrders(parsed *parser.OrderByClause, tableName, alias string) ([]ColumnRef, error) {
	if parsed == nil {
		return nil, nil
	}

	orders := []ColumnRef{}

	for _, item := range parsed.Items {
		ref := parser.ColumnAlias(item.Column)

		table := ref.Table()
		column := ref.Column()

		if table == alias {
			table = tableName
		}

		orders = append(orders, ColumnRef{table: table, column: column})
	}

	return orders, nil
}

func resolveProjection(items []parser.SelectItem) ([]ProjectedColumn, error) {
	projection := []ProjectedColumn{}

	for _, item := range items {
		if item.Star {
			continue
		}

		col, err := columnName(item.Expr)

		if err != nil {
			return nil, err
		}

		projection = append(projection, ProjectedColumn{source: col, alias: item.Alias})
	}

	return projection, nil
}

func columnName(expr *parser.Expr) (string, error) {
	if expr == nil || expr.Left == nil {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	if expr.Op != "" || expr.Right != nil {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	term := expr.Left

	if term.Op != "" || term.Right != nil {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	factor := term.Left

	if factor == nil || factor.Star || factor.Func != nil || factor.SubExpr != nil {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	if factor.Num != "" || factor.Hex != "" || factor.Str != "" {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	if factor.Column == "" {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	if _, after, ok := strings.Cut(factor.Column, "."); ok {
		return after, nil
	}

	return factor.Column, nil
}

func resolveCreate(parsed *parser.CreateStmt) (*ExecStmt, error) {
	columns := make([]Column, len(parsed.Columns))
	for i, c := range parsed.Columns {
		dt, err := parseDataType(c.Type)

		if err != nil {
			return nil, err
		}

		columns[i] = Column{name: c.Name, dataType: dt}
	}
	execStmt := &ExecStmt{
		createStmt: &CreateStmt{
			tableName: parsed.TableName,
			columns:   columns,
		},
	}
	return execStmt, nil
}

func resolveInsert(parsed *parser.InsertStmt) (*ExecStmt, error) {
	values := []any{}

	for _, parsedValue := range parsed.Values {
		val, err := toAny(parsedValue)

		if err != nil {
			return nil, err
		}

		values = append(values, val)
	}

	execStmt := &ExecStmt{
		insertStmt: &InsertStmt{
			tableName: parsed.TableName,
			values:    values,
		},
	}
	return execStmt, nil
}

func resolvePragma(parsed *parser.PragmaStmt) (*PragmaStmt, error) {
	property, err := pragma.Parse(parsed.Property)

	if err != nil {
		return nil, err
	}

	return &PragmaStmt{
		property: property,
		value:    parsed.Value,
	}, nil
}

func toAny(parsed parser.Value) (any, error) {
	switch {
	case parsed.Arg != "":
		return "?", nil

	case parsed.Str != "":
		return unquoteString(parsed.Str), nil

	case parsed.Num != "":
		if strings.Contains(parsed.Num, ".") {
			v, err := strconv.ParseFloat(parsed.Num, 64)
			if err != nil {
				return nil, err
			}
			return v, nil
		}

		v, err := strconv.ParseInt(parsed.Num, 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case parsed.Bytes != "":
		hexString := strings.TrimPrefix(parsed.Bytes, "0x")
		return hex.DecodeString(hexString)

	case parsed.Bool != "":
		switch parsed.Bool {
		case "TRUE":
			return true, nil
		case "FALSE":
			return false, nil
		}
		return nil, fmt.Errorf("invalid boolean %q", parsed.Bool)

	case parsed.Uuid != "":
		u, err := uuid.Parse(parsed.Uuid)
		if err != nil {
			return nil, err
		}
		return u, nil

	case parsed.Null != "":
		return nil, nil
	}

	return nil, fmt.Errorf("unsupported value %#v", parsed)
}

func unquoteString(s string) string {
	if len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}
