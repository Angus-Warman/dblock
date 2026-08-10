package internal

import (
	"fmt"

	"github.com/Angus-Warman/dblock/internal/codec"
)

type DataType uint8

const (
	NullType DataType = iota + 1
	AnyType
	IntegerType
	RealType
	TextType
	BoolType
	BlobType
	TimeType
	UUIDType
)

func PutDataType(e *codec.Encoder, dt DataType) {
	e.PutUint8(uint8(dt))
}

func GetDataType(d *codec.Decoder) (DataType, error) {
	b, err := d.GetUint8()

	if err != nil {
		return DataType(0), err
	}

	if b < uint8(NullType) || b > uint8(UUIDType) {
		return DataType(0), fmt.Errorf("GetDataType: byte %v outside of valid range", b)
	}

	return DataType(b), nil
}

func parseDataType(typeName string) (DataType, error) {
	switch typeName {
	case "ANY":
		return AnyType, nil
	case "TEXT":
		return TextType, nil
	case "INTEGER":
		return IntegerType, nil
	case "REAL":
		return RealType, nil
	case "BLOB":
		return BlobType, nil
	case "BOOL":
		return BoolType, nil
	case "TIME":
		return TimeType, nil
	case "UUID":
		return UUIDType, nil
	}

	return DataType(0), fmt.Errorf("parseDataType: unknown type %q", typeName)
}

func (dt DataType) String() string {
	switch dt {
	case AnyType:
		return "ANY"
	case NullType:
		return "NULL"
	case IntegerType:
		return "INTEGER"
	case RealType:
		return "REAL"
	case TextType:
		return "TEXT"
	case BoolType:
		return "BOOL"
	case BlobType:
		return "BLOB"
	case TimeType:
		return "TIME"
	case UUIDType:
		return "UUID"
	}

	return "unknown"
}
