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

	projection, err := resolveProjection(parsed.List.Items)

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

	where, err := resolveWhere(parsed.Where)

	if err != nil {
		return nil, err
	}

	sel := &SelectStmt{
		tableName:  parsed.TableName,
		projection: projection,
		joins:      joins,
		orders:     orders,
		where:      where,
	}

	queryStmt := &QueryStmt{
		selectStmt: sel,
	}

	return queryStmt, nil
}

func resolveWhere(w *parser.WhereClause) (*Expr, error) {
	if w == nil {
		return nil, nil
	}

	return resolveExpr(&w.Expr)
}

func parseJoinMode(s string) (JoinMode, error) {
	mode := JoinMode(s)

	switch mode {
	case InnerJoin, LeftOuterJoin, RightOuterJoin, FullOuterJoin, CrossJoin:
		return mode, nil
	case BareJoin:
		return InnerJoin, nil
	case LeftJoin:
		return LeftOuterJoin, nil
	case RightJoin:
		return RightOuterJoin, nil
	case FullJoin:
		return FullOuterJoin, nil
	}

	return mode, fmt.Errorf("join: could not parse %q as join type", s)
}

func resolveJoins(parsed []parser.JoinClause) ([]JoinStmt, error) {
	joins := []JoinStmt{}

	for _, j := range parsed {
		if j.On == nil {
			return nil, fmt.Errorf("join without ON clause")
		}

		mode, err := parseJoinMode(j.Mode)

		if err != nil {
			return nil, err
		}

		onExpr, err := resolveExpr(&j.On.Expr)

		if err != nil {
			return nil, fmt.Errorf("join: %w", err)
		}

		joins = append(joins, JoinStmt{
			tableName: j.Table,
			mode:      mode,
			onExpr:    onExpr,
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

		if err == nil {
			projection = append(projection, ProjectedColumn{source: col, alias: item.Alias})
			continue
		}

		expr, rerr := resolveExpr(item.Expr)

		if rerr != nil {
			return nil, rerr
		}

		projection = append(projection, ProjectedColumn{expr: expr, alias: item.Alias})
	}

	return projection, nil
}

func columnName(expr *parser.Expr) (string, error) {
	if expr == nil || len(expr.Factors) != 1 {
		return "", fmt.Errorf("unsupported expression in select list")
	}

	factor := expr.Factors[0]

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
