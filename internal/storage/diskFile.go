package storage

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type DiskFile struct {
	f *os.File
}

func OpenDiskFile(path string) (*DiskFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		if err == unix.EWOULDBLOCK {
			return nil, fmt.Errorf("OpenDiskFile: %s is locked by another process", path)
		}
		return nil, fmt.Errorf("OpenDiskFile: flock failed: %w", err)
	}

	return &DiskFile{f: f}, nil
}

func (d *DiskFile) Close() error {
	// Unlocking is optional, closing the fd releases the flock lock
	err := unix.Flock(int(d.f.Fd()), unix.LOCK_UN)

	if err != nil {
		return err
	}

	return d.f.Close()
}

func (f *DiskFile) Exists() (bool, error) {
	_, err := f.f.Stat()
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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
