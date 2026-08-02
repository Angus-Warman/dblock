package storage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestWal(t *testing.T) (*WalStorage, *MemoryFile) {
	t.Helper()
	main := OpenMemoryFile(t.Name() + "-main")
	wal := OpenMemoryFile(t.Name() + "-wal")
	store, err := NewWalStorage(main, wal)
	require.NoError(t, err)
	return store, main
}

func seedBlock(t *testing.T, main File, id BlockID, data []byte) {
	t.Helper()
	require.Len(t, data, BlockSize)
	start, _ := pageOffset(id)
	_, err := main.WriteAt(data, int64(start))
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

	txA := pager.NewTxStorage()
	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xAA), got)

	txA.PutBlock(0, testBlock(0xBB))
	require.NoError(t, txA.Commit())

	txB := pager.NewTxStorage()
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

	txA := pager.NewTxStorage()
	txB := pager.NewTxStorage()

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

	txC := pager.NewTxStorage()
	got, err = txC.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xCC), got)
}

// A TxStorage must not see blocks that were committed after it was opened.
func TestTxStorageDoesNotSeeNewBlocks(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	seedBlock(t, main, 0, testBlock(0xAA))

	txA := pager.NewTxStorage()
	_, err := txA.GetBlock(1)
	require.ErrorIs(t, err, ErrEmptyBlock)

	// Another TxStorage grows the database past txA's snapshot.
	txB := pager.NewTxStorage()
	txB.PutBlock(1, testBlock(0xBB))
	require.NoError(t, txB.Commit())

	// txA must still not see block 1.
	_, err = txA.GetBlock(1)
	require.ErrorIs(t, err, ErrEmptyBlock)
}

// Checkpoint must not break an active TxStorage's snapshot, and once the
// active TxStorage is gone it must move the new version into main.
func TestCheckpointPreservesSnapshots(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	seedBlock(t, main, 0, testBlock(0xAA))

	txA := pager.NewTxStorage()
	txB := pager.NewTxStorage()

	// txB commits a new version of block 0 while txA is still active.
	txB.PutBlock(0, testBlock(0xBB))
	require.NoError(t, txB.Commit())

	// Checkpoint must not copy the new version into main, or txA's snapshot
	// would change.
	require.NoError(t, pager.Checkpoint())

	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xAA), got)

	// Once txA is done, checkpointing reclaims the WAL and txC sees the new
	// version from main.
	require.NoError(t, txA.Commit())
	require.NoError(t, pager.Checkpoint())

	txC := pager.NewTxStorage()
	got, err = txC.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, testBlock(0xBB), got)
}
