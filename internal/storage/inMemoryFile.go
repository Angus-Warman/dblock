package storage

import (
	"io"
	"os"
	"sync"
)

var (
	memFileLock sync.Mutex
	memFileRefs map[string]int           = make(map[string]int)
	memFiles    map[string]*InMemoryFile = make(map[string]*InMemoryFile)
)

type InMemoryFile struct {
	name string
	data []byte
}

func OpenInMemoryFile(name string) *InMemoryFile {
	memFileLock.Lock()
	defer memFileLock.Unlock()

	existing, ok := memFiles[name]

	if ok {
		memFileRefs[name]++
		return existing
	}

	newFile := &InMemoryFile{
		name: name,
	}
	memFiles[name] = newFile
	memFileRefs[name] = 1

	return newFile
}

func (f *InMemoryFile) ReadAt(buf []byte, offset int64) (int, error) {
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

func (f *InMemoryFile) WriteAt(buf []byte, offset int64) (int, error) {
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

func (f *InMemoryFile) Truncate(size int64) error {
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

func (f *InMemoryFile) Size() (int64, error) {
	return int64(len(f.data)), nil
}

func (f *InMemoryFile) Sync() error {
	return nil
}

func (f *InMemoryFile) Close() error {
	memFileLock.Lock()
	defer memFileLock.Unlock()

	memFileRefs[f.name]--

	if memFileRefs[f.name] < 1 {
		delete(memFiles, f.name)
	}

	return nil
}
