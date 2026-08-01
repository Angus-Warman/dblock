package internal

import (
	"testing"

	"dblock2/internal/parser"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestToAny(t *testing.T) {
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	tests := []struct {
		name     string
		value    parser.Value
		expected any
	}{
		{name: "arg", value: parser.Value{Arg: "?"}, expected: "?"},
		{name: "single quoted string", value: parser.Value{Str: "'bar'"}, expected: "bar"},
		{name: "double quoted string", value: parser.Value{Str: `"bar"`}, expected: "bar"},
		{name: "backtick string", value: parser.Value{Str: "`bar`"}, expected: "bar"},
		{name: "empty string", value: parser.Value{Str: "''"}, expected: ""},
		{name: "integer", value: parser.Value{Num: "42"}, expected: int64(42)},
		{name: "real", value: parser.Value{Num: "3.14"}, expected: float64(3.14)},
		{name: "bytes", value: parser.Value{Bytes: "0xDEADBEEF"}, expected: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{name: "true", value: parser.Value{Bool: "TRUE"}, expected: true},
		{name: "false", value: parser.Value{Bool: "FALSE"}, expected: false},
		{name: "null", value: parser.Value{Null: "NULL"}, expected: nil},
		{name: "uuid", value: parser.Value{Uuid: validUUID.String()}, expected: validUUID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toAny(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestToAnyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		value parser.Value
	}{
		{name: "bad number", value: parser.Value{Num: "abc"}},
		{name: "bad hex", value: parser.Value{Bytes: "0x1"}},
		{name: "bad bool", value: parser.Value{Bool: "MAYBE"}},
		{name: "bad uuid", value: parser.Value{Uuid: "not-a-uuid"}},
		{name: "empty value", value: parser.Value{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := toAny(tt.value)
			require.Error(t, err)
		})
	}
}
