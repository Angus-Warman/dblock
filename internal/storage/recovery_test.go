package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"
)

var errInjected = errors.New("errorFile: injected error")

type errorFile struct {
	f       *MemoryFile
	errorIn int
}

// check decrements the operation counter and reports whether the next
// operation is allowed to proceed.
func (f *errorFile) check() bool {
	if f.errorIn <= 0 {
		return false
	}
	f.errorIn--
	return true
}

func openErrorFile(name string, errorAfter int) *errorFile {
	newFile := &errorFile{
		f:       OpenMemoryFile(name),
		errorIn: errorAfter,
	}

	return newFile
}

func (f *errorFile) ReadAt(buf []byte, offset int64) (int, error) {
	if !f.check() {
		return 0, errInjected
	}
	return f.f.ReadAt(buf, offset)
}

func (f *errorFile) WriteAt(buf []byte, offset int64) (int, error) {
	if !f.check() {
		n, _ := f.f.WriteAt(buf[:len(buf)/2], offset)
		return n, errInjected
	}
	return f.f.WriteAt(buf, offset)
}

func (f *errorFile) Truncate(size int64) error {
	if !f.check() {
		cur, _ := f.f.Size()
		f.f.Truncate((cur + size) / 2)
		return errInjected
	}
	return f.f.Truncate(size)
}

func (f *errorFile) Size() (int64, error) {
	if !f.check() {
		return 0, errInjected
	}
	return f.f.Size()
}

func (f *errorFile) Sync() error {
	if !f.check() {
		return errInjected
	}
	return f.f.Sync()
}

// Close deletes the file if it is the last reference to it.
func (f *errorFile) Close() error {
	if !f.check() {
		return errInjected
	}
	return nil
}

func TestCheckpointErrorMidway(t *testing.T) {
	main := openErrorFile(t.Name()+"-main", 1000)
	wal := openErrorFile(t.Name()+"-wal", 1000)

	store, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	addBlock(t, main, 0, 0xAA)

	txA := store.NewTxStorage()
	got, err := txA.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xAA), got)

	txA.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, txA.Commit())

	txB := store.NewTxStorage()
	got, err = txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)

	// "Oh no": the checkpoint's first main operation is the WriteAt that
	// copies the committed block into main. With main.errorIn = 0 that write
	// fails, so errorFile writes only half the block before reporting the
	// error.
	main.errorIn = 0

	err = store.Checkpoint()
	require.ErrorIs(t, err, errInjected)

	// The checkpoint never completed and the torn write left block 0 in main
	// corrupted: the first half holds the new version, the second half the old.
	main.errorIn = 1000

	torn := make([]byte, BlockSize)
	copy(torn, newBlock(0, 0xBB)[:BlockSize/2])
	copy(torn[BlockSize/2:], newBlock(0, 0xAA)[BlockSize/2:])

	got = make([]byte, BlockSize)
	_, err = main.ReadAt(got, 0)
	require.NoError(t, err)
	require.Equal(t, torn, got)
	require.NotEqual(t, newBlock(0, 0xBB), got)
	require.NotEqual(t, newBlock(0, 0xAA), got)

	// The intact committed version is still served from the WAL and the
	// file-change counter was never bumped.
	got, err = txB.GetBlock(0)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)

	meta, err := store.GetMetadata()
	require.NoError(t, err)
	require.Equal(t, uint32(0), meta.FileChangeCounter)
}

// makeFrame builds a single WAL frame: header (blockID, commit marker,
// checksum) followed by a full block of data, as PutBlocks writes it.
func makeFrame(id BlockID, commit bool, value byte) []byte {
	header := make([]byte, frameHeaderSize)
	binary.BigEndian.PutUint64(header[0:8], uint64(id))
	if commit {
		binary.BigEndian.PutUint64(header[8:16], 1)
	}
	buf := bytes.Repeat([]byte{value}, BlockSize)
	binary.BigEndian.PutUint64(header[16:24], uint64(crc32.ChecksumIEEE(buf)))
	return append(header, buf...)
}

// A crash that tears the WAL mid-transaction must not lose committed data:
// Recover replays every frame up to the last valid commit and discards the
// torn transaction that follows it.
func TestRecoverIgnoresTornTransaction(t *testing.T) {
	main := OpenMemoryFile(t.Name() + "-main")
	wal := OpenMemoryFile(t.Name() + "-wal")

	addBlock(t, main, 0, 0xAA)

	store, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	tx := store.NewTxStorage()
	tx.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, tx.Commit())

	tx = store.NewTxStorage()
	tx.PutBlock(1, newBlock(1, 0xCC))
	require.NoError(t, tx.Commit())

	// Simulate a crash mid-commit: a frame whose payload is torn, so its
	// stored checksum no longer matches the data.
	torn := makeFrame(2, true, 0xDD)
	torn[frameHeaderSize] ^= 0xFF
	_, err = wal.WriteAt(torn, int64(store.walEnd))
	require.NoError(t, err)

	// Reopening replays the WAL and recovers both commits, ignoring the torn
	// frame that follows them.
	reopened, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	got, err := reopened.GetBlock(0, reopened.maxWalID)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)

	got, err = reopened.GetBlock(1, reopened.maxWalID)
	require.NoError(t, err)
	require.Equal(t, newBlock(1, 0xCC), got)

	// Nothing past the last valid commit is indexed, walEnd points at the end
	// of that commit, and the two recovered frames own WalIDs 0 and 1.
	_, ok := reopened.walIndex[2]
	require.False(t, ok)
	require.Equal(t, BlockOffset(2*frameSize), reopened.walEnd)
	require.Equal(t, WalID(2), reopened.maxWalID)
}

// Frames appended after the last commit but before a crash - a dangling
// uncommitted transaction - must not become visible on recovery.
func TestRecoverDiscardsUncommittedFrames(t *testing.T) {
	main := OpenMemoryFile(t.Name() + "-main")
	wal := OpenMemoryFile(t.Name() + "-wal")

	store, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	tx := store.NewTxStorage()
	tx.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, tx.Commit())

	// A valid frame for block 1 that never received a commit marker before
	// the crash.
	_, err = wal.WriteAt(makeFrame(1, false, 0xCC), int64(store.walEnd))
	require.NoError(t, err)

	reopened, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	got, err := reopened.GetBlock(0, reopened.maxWalID)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)

	_, ok := reopened.walIndex[1]
	require.False(t, ok)
	require.Equal(t, BlockOffset(frameSize), reopened.walEnd)
}

// A torn checkpoint may leave a half-written block in main; recovery must
// still serve the intact committed version from the WAL.
func TestRecoverRestoresIntactVersionAfterTornCheckpoint(t *testing.T) {
	main := openErrorFile(t.Name()+"-main", 1000)
	wal := openErrorFile(t.Name()+"-wal", 1000)

	store, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	addBlock(t, main, 0, 0xAA)

	tx := store.NewTxStorage()
	tx.PutBlock(0, newBlock(0, 0xBB))
	require.NoError(t, tx.Commit())

	// Crash mid-checkpoint: main's copy of block 0 is left torn.
	main.errorIn = 0
	err = store.Checkpoint()
	require.ErrorIs(t, err, errInjected)

	// Reopening replays the WAL, which still holds the intact committed
	// version.
	main.errorIn = 1000
	wal.errorIn = 1000

	reopened, err := NewWalStorage(main, wal)
	require.NoError(t, err)

	got, err := reopened.GetBlock(0, reopened.maxWalID)
	require.NoError(t, err)
	require.Equal(t, newBlock(0, 0xBB), got)
}
