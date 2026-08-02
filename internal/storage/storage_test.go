package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	wal, err := OpenWalStorage(":memory:")
	require.NoError(t, err)
	require.NotNil(t, wal)
}

func newTestWal(t *testing.T) (*WalStorage, *InMemoryFile) {
	t.Helper()
	main := OpenInMemoryFile(t.Name() + "-main")
	wal := OpenInMemoryFile(t.Name() + "-wal")
	return NewWalPager(main, wal), main
}

func seedBlock(t *testing.T, main StorageFile, id BlockID, data []byte) {
	t.Helper()
	require.Len(t, data, BlockSize)
	_, err := main.WriteAt(data, int64(id*BlockSize))
	require.NoError(t, err)
}

func testBlock(seed byte) []byte {
	return bytes.Repeat([]byte{seed}, BlockSize)
}

// The main purpose of WalStorage: a committed block is written to the WAL and
// becomes visible to a new TxStorage opened afterwards.
func TestWalStorageCommitVisibleToNewTx(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	seedBlock(t, main, 0, testBlock(0xAA))

	txA := pager.NewTxPager()
	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xAA), got)

	txA.PutBlock(0, testBlock(0xBB))
	require.NoError(t, txA.Commit())

	txB := pager.NewTxPager()
	got, err = txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xBB), got)
}

// The main purpose of TxStorage: while two TxStorage are active, commits from
// one must not affect reads from the other.
func TestTwoTxStoragesAreIsolated(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	seedBlock(t, main, 0, testBlock(0xAA))

	txA := pager.NewTxPager()
	txB := pager.NewTxPager()

	// Both see the same snapshot before anything is written.
	for _, tx := range []*TxStorage{txA, txB} {
		got, err := tx.GetBlock(0)
		require.NoError(t, err)
		require.Equal(t, testBlock(0xAA), got)
	}

	// txA commits a new version of block 0.
	txA.PutBlock(0, testBlock(0xBB))
	require.NoError(t, txA.Commit())

	// txB, still active, must keep seeing the original snapshot.
	got, err := txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xAA), got)

	// txB can commit too; a fresh TxStorage then sees the latest version.
	txB.PutBlock(0, testBlock(0xCC))
	require.NoError(t, txB.Commit())

	txC := pager.NewTxPager()
	got, err = txC.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xCC), got)
}

// A TxStorage must not see blocks that were committed after it was opened.
func TestTxStorageDoesNotSeeNewBlocks(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	seedBlock(t, main, 0, testBlock(0xAA))

	txA := pager.NewTxPager()
	_, err := txA.GetBlock(1)
	require.ErrorIs(t, err, ErrEmptyBlock)

	// Another TxStorage grows the database past txA's snapshot.
	txB := pager.NewTxPager()
	txB.PutBlock(1, testBlock(0xBB))
	require.NoError(t, txB.Commit())

	// txA must still not see block 1.
	_, err = txA.GetBlock(1)
	require.ErrorIs(t, err, ErrEmptyBlock)
}
