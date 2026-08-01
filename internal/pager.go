package internal

import (
	"fmt"
	"sync"
)

type Pager interface {
	GetPage(PageID) (*Page, error)
	PutPage(PageID, *Page) error
	NextID() PageID
	Commit() error
	Rollback() error
	Close() error
}

type MemoryPager struct {
	file     []byte
	wal      []byte
	nextID   PageID
	baseNext PageID
}

// pageOffset returns the byte range of the page with the given ID.
func pageOffset(id PageID) (int, int) {
	start := int(id) * PageSize
	return start, start + PageSize
}

// ensure grows the slice so the page with the given ID fits.
func ensurePage(buf []byte, id PageID) []byte {
	_, end := pageOffset(id)
	if end > len(buf) {
		next := make([]byte, end)
		copy(next, buf)
		return next
	}
	return buf
}

// NextID implements [Pager].
func (m *MemoryPager) NextID() PageID {
	id := m.nextID
	m.nextID++
	return id
}

// PutPage implements [Pager].
func (m *MemoryPager) PutPage(id PageID, page *Page) error {
	data, err := page.Encode()
	if err != nil {
		return err
	}
	m.wal = ensurePage(m.wal, id)
	start, end := pageOffset(id)
	copy(m.wal[start:end], data)
	return nil
}

// GetPage implements [Pager].
func (m *MemoryPager) GetPage(id PageID) (*Page, error) {
	start, end := pageOffset(id)
	if end > len(m.file) {
		return nil, ErrEmptyPage
	}
	return Decode(m.file[start:end])
}

// Commit implements [Pager].
func (m *MemoryPager) Commit() error {
	m.file = append([]byte{}, m.wal...) // TODO should only replace dirty pages
	m.baseNext = m.nextID
	return nil
}

// Rollback implements [Pager].
func (m *MemoryPager) Rollback() error {
	m.wal = append([]byte{}, m.file...)
	m.nextID = m.baseNext
	return nil
}

var (
	memDBsMu       sync.Mutex
	globalMemPager *MemoryPager
	memDBRefs      = 0
)

func NewPager(dsn string) (Pager, error) {
	if dsn == ":memory:" {
		memDBsMu.Lock()
		defer memDBsMu.Unlock()

		if globalMemPager == nil {
			p := NewMemoryPager()
			globalMemPager = p
		}

		memDBRefs++
		return globalMemPager, nil
	}

	return nil, fmt.Errorf("WIP")
}

func (p *MemoryPager) Close() error {
	memDBsMu.Lock()
	defer memDBsMu.Unlock()

	memDBRefs--
	if memDBRefs > 0 {
		return nil
	}

	globalMemPager = nil
	return nil
}

func NewMemoryPager() *MemoryPager {
	return &MemoryPager{
		nextID: 1, // RootSchemaPageID (0) is reserved
	}
}
