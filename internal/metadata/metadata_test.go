package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundtrip(t *testing.T) {
	original := New()
	buf := original.Encode()
	decoded, err := Decode(buf)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}
