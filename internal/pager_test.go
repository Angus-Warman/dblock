package internal

import (
	"testing"

	"dblock2/internal/storage"

	"github.com/stretchr/testify/require"
)

func newTestWal(t *testing.T) *storage.WalStorage {
	t.Helper()
	main := storage.OpenMemoryFile(t.Name() + "-main")
	wal := storage.OpenMemoryFile(t.Name() + "-wal")
	store, err := storage.NewWalStorage(main, wal)
	require.NoError(t, err)
	return store
}

func TestStoragePagerRoundTrip(t *testing.T) {
	src := &PagerSource{wal: newTestWal(t)}
	pager := src.Begin()

	require.Equal(t, PageID(1), pager.NextID())
	require.Equal(t, PageID(2), pager.NextID())

	_, err := pager.GetPage(5)
	require.ErrorIs(t, err, ErrEmptyPage)

	leaf := &Page{
		ID:      1,
		IsLeaf:  true,
		NumKeys: 1,
		Keys:    [][]byte{[]byte("k")},
		Values:  [][]byte{[]byte("v")},
	}

	require.NoError(t, pager.PutPage(1, leaf))

	got, err := pager.GetPage(1)
	require.NoError(t, err)
	require.Equal(t, leaf.Keys, got.Keys)
	require.Equal(t, leaf.Values, got.Values)

	require.NoError(t, pager.Close())
}
