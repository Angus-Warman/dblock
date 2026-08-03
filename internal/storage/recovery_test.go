package storage

import (
	"errors"
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
