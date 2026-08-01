package internal

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Row struct {
	Values []any
}

func (r *Row) Encode() []byte {
	e := NewEncoder()

	e.PutAsUint16(len(r.Values))

	for _, rv := range r.Values {
		EncodeValue(e, rv)
	}

	return e.Bytes()
}

func EncodeValue(e *Encoder, val any) {
	if val == nil {
		e.PutDataType(NullType)
		return
	}

	switch v := val.(type) {
	case int64:
		e.PutDataType(IntegerType)
		e.PutInt64(v)
	case float64:
		e.PutDataType(RealType)
		e.PutFloat64(v)
	case string:
		e.PutDataType(TextType)
		e.PutStringWithLength(v)
	case bool:
		e.PutDataType(BoolType)
		e.PutBool(v)
	case []byte:
		e.PutDataType(BlobType)
		e.PutBytesWithLength(v)
	case time.Time:
		e.PutDataType(TimeType)
		_, offset := v.Zone()
		e.PutInt64(v.Unix())
		e.PutAsUint32(v.Nanosecond())
		e.PutInt16(int16(offset / 60))
	case uuid.UUID:
		e.PutDataType(UUIDType)
		e.PutBytes(v[:])
	default:
		panic(fmt.Errorf("no encoding available for %#v %T", val, val))
	}
}

func DecodeRow(buf []byte) (Row, error) {
	var zero Row

	d := NewDecoder(buf)

	count, err := d.GetUint16()
	if err != nil {
		return zero, fmt.Errorf("row too short: %w", err)
	}

	values := make([]any, count)

	for i := range count {
		dt, err := d.GetDataType()
		if err != nil {
			return zero, err
		}

		switch DataType(dt) {
		case NullType:
			values[i] = nil

		case IntegerType:
			v, err := d.GetInt64()
			if err != nil {
				return zero, fmt.Errorf("truncated integer: %w", err)
			}
			values[i] = v

		case RealType:
			v, err := d.GetFloat64()
			if err != nil {
				return zero, fmt.Errorf("truncated real: %w", err)
			}
			values[i] = v

		case TextType:
			v, err := d.GetStringWithLength()
			if err != nil {
				return zero, fmt.Errorf("truncated text: %w", err)
			}
			values[i] = v

		case BoolType:
			v, err := d.GetUint8()
			if err != nil {
				return zero, fmt.Errorf("truncated bool: %w", err)
			}
			values[i] = v != 0

		case BlobType:
			v, err := d.GetBytesWithLength()
			if err != nil {
				return zero, fmt.Errorf("truncated blob: %w", err)
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
			return zero, fmt.Errorf("unsupported datatype %d", dt)
		}
	}

	return Row{Values: values}, nil
}
