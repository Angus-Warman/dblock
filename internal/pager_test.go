package internal

import (
	"testing"

	"dblock2/internal/metadata"
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
	pager, err := src.Begin()
	require.NoError(t, err)

	require.Equal(t, PageID(1), pager.NextID())
	require.Equal(t, PageID(2), pager.NextID())

	_, err = pager.GetPage(5)
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

func TestPagerMetadataRoundTrip(t *testing.T) {
	src := &PagerSource{wal: newTestWal(t)}
	pager, err := src.Begin()
	require.NoError(t, err)

	// A fresh database reports default metadata.
	m, err := pager.GetMetadata()
	require.NoError(t, err)
	require.Equal(t, metadata.New(), m)

	// Writing the root schema page must not clobber the metadata.
	leaf := &Page{
		ID:      RootSchemaPageID,
		IsLeaf:  true,
		NumKeys: 1,
		Keys:    [][]byte{[]byte("k")},
		Values:  [][]byte{[]byte("v")},
	}
	require.NoError(t, pager.PutPage(RootSchemaPageID, leaf))

	// Updating the metadata must not clobber the root page's data.
	m.SchemaVersion = 7
	m.FileVersion = 3
	m.CalculateChecksum()
	require.NoError(t, pager.PutMetadata(m))

	got, err := pager.GetMetadata()
	require.NoError(t, err)
	require.Equal(t, m, got)

	page, err := pager.GetPage(RootSchemaPageID)
	require.NoError(t, err)
	require.Equal(t, leaf.Keys, page.Keys)
	require.Equal(t, leaf.Values, page.Values)

	require.NoError(t, pager.Commit())
	require.NoError(t, pager.Close())
}
