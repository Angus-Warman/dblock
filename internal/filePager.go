package internal

import (
	"os"
)

func NewMemoryPager() (*MemoryPager, error) {
	return &MemoryPager{
		nextID: 1, // RootSchemaPageID (0) is reserved
	}, nil
}

type FilePager struct {
	file   *os.File
	size   int64
	nextID PageID
}

// PutPage implements [Pager].
func (f *FilePager) PutPage(id PageID, page *Page) error {
	data, err := page.Encode()

	if err != nil {
		return err
	}

	_, end := pageOffset(id)

	if int64(end) > f.size {
		if err := f.file.Truncate(int64(end)); err != nil {
			return err
		}

		f.size = int64(end)
	}

	_, err = f.file.WriteAt(data, int64(id)*PageSize)

	return err
}

// GetPage implements [Pager].
func (f *FilePager) GetPage(id PageID) (*Page, error) {
	start := int64(id) * PageSize

	if start+PageSize > f.size {
		return nil, ErrEmptyPage
	}

	buf := make([]byte, PageSize)

	_, err := f.file.ReadAt(buf, start)

	if err != nil {
		return nil, err
	}

	return Decode(buf)
}

// NextID implements [Pager].
func (f *FilePager) NextID() PageID {
	id := f.nextID
	f.nextID++
	return id
}

// Close implements [Pager].
func (f *FilePager) Close() error {
	return f.file.Close()
}

func NewFilePager(dsn string) (*FilePager, error) {
	f, err := os.OpenFile(dsn, os.O_RDWR|os.O_CREATE, 0o644)

	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()

	if err != nil {
		f.Close()
		return nil, err
	}

	size := stat.Size()

	// Page IDs are allocated sequentially, so the next free page is the
	// first one past the end of the file. Never below 1 (RootSchemaPageID
	// is reserved).
	nextID := PageID((size + PageSize - 1) / PageSize)

	if nextID < 1 {
		nextID = 1
	}

	return &FilePager{
		file:   f,
		size:   size,
		nextID: nextID,
	}, nil
}
