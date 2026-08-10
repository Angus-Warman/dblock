package storage

import (
	"bytes"
	"testing"

	"github.com/Angus-Warman/dblock/internal/metadata"

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

func newBlock(id BlockID, value byte) []byte {
	buf := bytes.Repeat([]byte{value}, BlockSize)

	if id == 0 {
		copy(buf[:metadata.Length], metadata.New().Encode())
	}

	return buf
}

func addBlock(t *testing.T, main File, id BlockID, value byte) {
	t.Helper()

	buf := newBlock(id, value)

	_, err := main.WriteAt(buf, int64(id*BlockSize))
	require.NoError(t, err)
}

// The main purpose of WalStorage: a committed block is written to the WAL and
// becomes visible to a new TxStorage opened afterwards.
func TestWalStorageCommitVisibleToNewTx(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	addBlock(t, main, 0, 0xAA)

	txA := pager.NewTxStorage()
	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xAA), got)

	txA.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, txA.Commit())

	txB := pager.NewTxStorage()
	got, err = txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)
}

// The main purpose of TxStorage: while two TxStorage are active, commits from
// one must not affect reads from the other.
func TestTwoTxStoragesAreIsolated(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	addBlock(t, main, 0, 0xAA)

	txA := pager.NewTxStorage()
	txB := pager.NewTxStorage()

	// Both see the same snapshot before anything is written.
	for _, tx := range []*TxStorage{txA, txB} {
		got, err := tx.GetBlock(0)
		require.NoError(t, err)
		require.Equal(t, newBlock(0, 0xAA), got)
	}

	// txA commits a new version of block 0.
	txA.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, txA.Commit())

	// txB, still active, must keep seeing the original snapshot.
	got, err := txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xAA), got)

	// txB can commit too; a fresh TxStorage then sees the latest version.
	txB.PutBlock(0, newBlock(0, 0xCC))
	require.NoError(t, txB.Commit())

	txC := pager.NewTxStorage()
	got, err = txC.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xCC), got)
}

// A TxStorage must not see blocks that were committed after it was opened.
func TestTxStorageDoesNotSeeNewBlocks(t *testing.T) {
	pager, main := newTestWal(t)
	defer pager.Close()

	addBlock(t, main, 0, 0xAA)

	txA := pager.NewTxStorage()
	_, err := txA.GetBlock(1)
	require.ErrorIs(t, err, ErrEmptyBlock)

	// Another TxStorage grows the database past txA's snapshot.
	txB := pager.NewTxStorage()
	txB.PutBlock(1, newBlock(1, 0xBB))
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

	addBlock(t, main, 0, 0xAA)

	txA := pager.NewTxStorage()
	txB := pager.NewTxStorage()

	// txB commits a new version of block 0 while txA is still active.
	txB.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, txB.Commit())

	// Checkpoint must not copy the new version into main, or txA's snapshot
	// would change.
	require.NoError(t, pager.Checkpoint())

	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xAA), got)

	// Once txA is done, checkpointing reclaims the WAL and txC sees the new
	// version from main. Moving pages into main also bumps the metadata's
	// file-change counter.
	require.NoError(t, txA.Commit())
	require.NoError(t, pager.Checkpoint())

	txC := pager.NewTxStorage()
	meta, err := txC.wal.GetMetadata()
	require.NoError(t, err)
	require.Equal(t, uint32(2), meta.FileVersion)
}
