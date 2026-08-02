package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	wal, err := OpenWalStorage(":memory:")
	require.NoError(t, err)
	require.NotNil(t, wal)
}
