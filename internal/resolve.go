package internal

import (
	"dblock2/internal/parser"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func resolveCreate(parsed *parser.CreateStmt) (*ExecStmt, error) {
	columns := make([]string, len(parsed.Columns))
	for i, c := range parsed.Columns {
		columns[i] = c.Name
	}
	execStmt := &ExecStmt{
		createStmt: &CreateStmt{
			tableName:   parsed.TableName,
			columnNames: columns,
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
