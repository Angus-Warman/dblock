//go:build windows

package storage

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

type DiskFile struct {
	f *os.File
}

// lockFileMax is used as the length for LockFileEx when locking "the whole
// file" — Windows byte-range locks need an explicit length, and since we
// don't know the eventual file size up front, we lock an effectively
// unbounded range (matches the whole-file semantics of flock on Linux).
const lockFileMax = ^uint32(0) // 0xFFFFFFFF

func OpenDiskFile(path string) (*DiskFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	handle := windows.Handle(f.Fd())

	ol := new(windows.Overlapped) // offset 0, locking from the start of the file

	err = windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockFileMax,
		lockFileMax,
		ol,
	)
	if err != nil {
		f.Close()
		if err == windows.ERROR_LOCK_VIOLATION {
			return nil, fmt.Errorf("OpenDiskFile: %s is locked by another process", path)
		}
		return nil, fmt.Errorf("OpenDiskFile: LockFileEx failed: %w", err)
	}

	return &DiskFile{f: f}, nil
}

func (d *DiskFile) Close() error {
	// Unlocking is optional; closing the handle releases the lock.
	// Being explicit for parity with the Linux implementation.
	handle := windows.Handle(d.f.Fd())
	ol := new(windows.Overlapped)

	err := windows.UnlockFileEx(handle, 0, lockFileMax, lockFileMax, ol)
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
