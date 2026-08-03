package internal

import (
	"dblock2/internal/codec"
	"fmt"
)

type DataType uint8

const (
	NullType DataType = iota + 1
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
