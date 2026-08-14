package internal

import (
	"fmt"
	"time"

	"github.com/Angus-Warman/dblock/internal/codec"

	"github.com/google/uuid"
)

type Row struct {
	ID     RowID
	Values []any
}

func (r *Row) Encode() []byte {
	e := codec.NewEncoder()

	e.PutAsUint16(len(r.Values))

	for _, rv := range r.Values {
		EncodeValue(e, rv)
	}

	return e.Bytes()
}

func EncodeValue(e *codec.Encoder, val any) {
	if val == nil {
		PutDataType(e, NullType)
		return
	}

	switch v := val.(type) {
	case int64:
		PutDataType(e, IntegerType)
		e.PutInt64(v)
	case float64:
		PutDataType(e, RealType)
		e.PutFloat64(v)
	case string:
		PutDataType(e, TextType)
		e.PutStringWithLength(v)
	case bool:
		PutDataType(e, BoolType)
		e.PutBool(v)
	case []byte:
		PutDataType(e, BlobType)
		e.PutBytesWithLength(v)
	case time.Time:
		PutDataType(e, TimeType)
		_, offset := v.Zone()
		e.PutInt64(v.Unix())
		e.PutAsUint32(v.Nanosecond())
		e.PutInt16(int16(offset / 60))
	case uuid.UUID:
		PutDataType(e, UUIDType)
		e.PutBytes(v[:])
	default:
		panic(fmt.Errorf("no encoding available for %#v %T", val, val))
	}
}

func DecodeRow(buf []byte) (Row, error) {
	var zero Row

	d := codec.NewDecoder("row", buf)

	count, err := d.GetUint16()
	if err != nil {
		return zero, fmt.Errorf("decode row: %w", err)
	}

	values := make([]any, count)

	for i := range count {
		dt, err := GetDataType(d)
		if err != nil {
			return zero, fmt.Errorf("decode row: %w", err)
		}

		switch dt {
		case NullType:
			values[i] = nil

		case IntegerType:
			v, err := d.GetInt64()
			if err != nil {
				return zero, fmt.Errorf("decode row: %w", err)
			}
			values[i] = v

		case RealType:
			v, err := d.GetFloat64()
			if err != nil {
				return zero, fmt.Errorf("decode row: %w", err)
			}
			values[i] = v

		case TextType:
			v, err := d.GetStringWithLength()
			if err != nil {
				return zero, fmt.Errorf("decode row: %w", err)
			}
			values[i] = v

		case BoolType:
			v, err := d.GetUint8()
			if err != nil {
				return zero, fmt.Errorf("decode row: %w", err)
			}
			values[i] = v != 0

		case BlobType:
			v, err := d.GetBytesWithLength()
			if err != nil {
				return zero, fmt.Errorf("decode row: %w", err)
			}
			values[i] = v

		case TimeType:
			sec, err := d.GetInt64()
			if err != nil {
				return zero, fmt.Errorf("truncated time sec: %w", err)
			}
			nsec, err := d.GetAsUint32()
			if err != nil {
				return zero, fmt.Errorf("truncated time nsec: %w", err)
			}
			offsetMin, err := d.GetInt16()
			if err != nil {
				return zero, fmt.Errorf("truncated time offset: %w", err)
			}
			var loc *time.Location
			if offsetMin == 0 {
				loc = time.UTC
			} else {
				loc = time.FixedZone("", int(offsetMin)*60)
			}
			values[i] = time.Unix(sec, int64(nsec)).In(loc)

		case UUIDType:
			b, err := d.GetBytes(16)
			if err != nil {
				return zero, fmt.Errorf("truncated uuid: %w", err)
			}
			var u uuid.UUID
			copy(u[:], b)
			values[i] = u

		default:
			return zero, fmt.Errorf("decode row: unsupported datatype %d", dt)
		}
	}

	return Row{Values: values}, nil
}

func (r *Row) GetValue(colMap map[string]int, colName string) (any, error) {
	i, ok := colMap[colName]

	if !ok {
		return nil, fmt.Errorf("column %q not found", colName)
	}

	if i < 0 || i >= len(r.Values) {
		return nil, fmt.Errorf("col ordinal %v out of bounds", i)
	}

	return r.Values[i], nil
}

func GetRowValueAs[T any](r *Row, colIdx int) (T, error) {
	var zero T

	if colIdx < 0 || colIdx >= len(r.Values) {
		return zero, fmt.Errorf("col ordinal %v out of bounds", colIdx)
	}

	raw := r.Values[colIdx]

	value, ok := raw.(T)

	if !ok {
		return zero, fmt.Errorf("value %v is not type %T", raw, zero)
	}

	return value, nil
}

func (r *Row) Clone() Row {
	cloned := Row{
		ID:     r.ID,
		Values: make([]any, len(r.Values)),
	}

	for i, v := range r.Values {
		if b1, ok := v.([]byte); ok {
			b2 := make([]byte, len(b1))
			copy(b2, b1)
			cloned.Values[i] = b2
		} else {
			cloned.Values[i] = v
		}
	}

	return cloned
}
