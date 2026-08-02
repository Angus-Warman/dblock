package storage

import (
	"io"
	"os"
)

type MemoryFile struct {
	name string
	data []byte
}

func OpenMemoryFile(name string) *MemoryFile {
	newFile := &MemoryFile{
		name: name,
	}

	return newFile
}

func (f *MemoryFile) ReadAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, os.ErrInvalid
	}
	if offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(buf, f.data[offset:])
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

func (f *MemoryFile) WriteAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, os.ErrInvalid
	}
	end := offset + int64(len(buf))
	if end > int64(len(f.data)) {
		grown := make([]byte, end)
		copy(grown, f.data)
		f.data = grown
	}
	n := copy(f.data[offset:end], buf)
	return n, nil
}

func (f *MemoryFile) Truncate(size int64) error {
	if size < 0 {
		return os.ErrInvalid
	}
	if size <= int64(len(f.data)) {
		f.data = f.data[:size]
		return nil
	}
	grown := make([]byte, size)
	copy(grown, f.data)
	f.data = grown
	return nil
}

func (f *MemoryFile) Size() (int64, error) {
	return int64(len(f.data)), nil
}

func (f *MemoryFile) Sync() error {
	return nil
}

// Close deletes the file if it is the last reference to it.
func (f *MemoryFile) Close() error {
	return nil
}
