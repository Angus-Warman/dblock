package internal

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func zeroValue(dt DataType) any {
	switch dt {
	case AnyType, NullType:
		return nil
	case IntegerType:
		return int64(0)
	case RealType:
		return float64(0)
	case TextType:
		return ""
	case BoolType:
		return false
	case BlobType:
		return []byte{}
	case TimeType:
		return time.Time{}
	case UUIDType:
		return uuid.UUID{}
	default:
		panic(fmt.Errorf("no zero value for DataType %v", dt))
	}
}

func coerceValue(col Column, value any) (any, error) {
	if value == nil {
		if col.dataType == AnyType {
			return nil, nil
		}

		return nil, fmt.Errorf("cannot insert null value into column %q, expects %v", col.name, col.dataType)
	}

	switch col.dataType {
	case AnyType:
		return value, nil

	case IntegerType:
		if v, ok := value.(int64); ok {
			return v, nil
		}

	case RealType:
		switch v := value.(type) {
		case float64:
			return v, nil
		case int64:
			return float64(v), nil
		}

	case TextType:
		switch v := value.(type) {
		case string:
			return v, nil
		case []byte:
			return string(v), nil
		case int64:
			return strconv.FormatInt(v, 10), nil
		case float64:
			return strconv.FormatFloat(v, 'g', -1, 64), nil
		case bool:
			return strconv.FormatBool(v), nil
		case time.Time:
			return v.Format(time.RFC3339Nano), nil
		}

	case BoolType:
		switch v := value.(type) {
		case bool:
			return v, nil
		case int64:
			switch v {
			case 1:
				return true, nil
			case 0:
				return false, nil
			}
		case string:
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q to bool", v)
			}
			return parsed, nil
		}

	case BlobType:
		switch v := value.(type) {
		case []byte:
			return v, nil
		case string:
			return []byte(v), nil
		}

	case TimeType:
		if v, ok := value.(time.Time); ok {
			return v, nil
		}

	case UUIDType:
		switch v := value.(type) {
		case uuid.UUID:
			return v.String(), nil
		case string:
			if _, err := uuid.Parse(v); err == nil {
				return v, nil
			}
		case []byte:
			if len(v) == 16 {
				return v, nil
			}
		}

	default:
		return nil, fmt.Errorf("unsupported data type %d", col.dataType)
	}

	return nil, fmt.Errorf("column %q expects %s, got %T(%v)", col.name, col.dataType, value, value)
}
