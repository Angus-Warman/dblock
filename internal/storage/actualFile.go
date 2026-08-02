package storage

import "os"

type StorageFile interface {
	ReadAt(buf []byte, offset int64) (int, error)
	WriteAt(buf []byte, offset int64) (int, error)
	Truncate(size int64) error
	Size() (int64, error)
	// Sync forces durability (fsync on disk). No-op for in-memory files.
	Sync() error
	Close() error
}

type ActualFile struct {
	f *os.File
}

func OpenActualFile(path string) (*ActualFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)

	if err != nil {
		return nil, err
	}

	return &ActualFile{f: f}, nil
}

func (a *ActualFile) ReadAt(buf []byte, offset int64) (int, error) {
	return a.f.ReadAt(buf, offset)
}

func (a *ActualFile) WriteAt(buf []byte, offset int64) (int, error) {
	return a.f.WriteAt(buf, offset)
}

func (a *ActualFile) Truncate(size int64) error {
	return a.f.Truncate(size)
}

func (a *ActualFile) Size() (int64, error) {
	info, err := a.f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (a *ActualFile) Sync() error {
	return a.f.Sync()
}

func (a *ActualFile) Close() error {
	return a.f.Close()
}
