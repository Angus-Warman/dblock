package storage

type File interface {
	Exists() (bool, error)
	ReadAt(buf []byte, offset int64) (int, error)
	WriteAt(buf []byte, offset int64) (int, error)
	Truncate(size int64) error
	Size() (int64, error)
	Sync() error
	Close() error
}
