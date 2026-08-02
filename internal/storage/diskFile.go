package storage

import "os"

type DiskFile struct {
	f *os.File
}

func OpenDiskFile(path string) (*DiskFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)

	if err != nil {
		return nil, err
	}

	return &DiskFile{f: f}, nil
}

func (f *DiskFile) ReadAt(buf []byte, offset int64) (int, error) {
	return f.f.ReadAt(buf, offset)
}

func (f *DiskFile) WriteAt(buf []byte, offset int64) (int, error) {
	return f.f.WriteAt(buf, offset)
}

func (f *DiskFile) Truncate(size int64) error {
	return f.f.Truncate(size)
}

func (f *DiskFile) Size() (int64, error) {
	info, err := f.f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (f *DiskFile) Sync() error {
	return f.f.Sync()
}

func (f *DiskFile) Close() error {
	return f.f.Close()
}
