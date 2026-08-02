package internal

import (
	"testing"

	"dblock2/internal/storage"

	"github.com/stretchr/testify/require"
)

func TestStoragePagerRoundTrip(t *testing.T) {
	main := storage.OpenInMemoryFile(t.Name() + "-main")
	wal := storage.OpenInMemoryFile(t.Name() + "-wal")

	src := &PagerSource{wal: storage.NewWalStorage(main, wal)}
	pager := src.Begin()

	// RootSchemaPageID (0) is reserved, so the first allocated page is 1.
	require.Equal(t, PageID(1), pager.NextID())
	require.Equal(t, PageID(2), pager.NextID())

	// Uninitialized pages read as empty.
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
