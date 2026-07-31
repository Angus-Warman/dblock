package internal

import "fmt"

type Pager interface {
	GetPage(PageID) (*Page, error)
	PutPage(PageID, *Page) error
	NextID() PageID
	Commit() error
	Rollback() error
}

type MemoryPager struct {
	pages     map[PageID][]byte
	committed map[PageID][]byte
	nextID    PageID
	baseNext  PageID
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
	m.pages[id] = data
	return nil
}

// GetPage implements [Pager].
func (m *MemoryPager) GetPage(id PageID) (*Page, error) {
	data, ok := m.pages[id]
	if !ok {
		return nil, ErrEmptyPage
	}
	return Decode(data)
}

// Commit implements [Pager].
func (m *MemoryPager) Commit() error {
	m.committed = clonePages(m.pages)
	m.baseNext = m.nextID
	return nil
}

// Rollback implements [Pager].
func (m *MemoryPager) Rollback() error {
	m.pages = clonePages(m.committed)
	m.nextID = m.baseNext
	return nil
}

func clonePages(src map[PageID][]byte) map[PageID][]byte {
	dst := make(map[PageID][]byte, len(src))
	for id, data := range src {
		dst[id] = append([]byte{}, data...)
	}
	return dst
}

func NewPager(dsn string) (Pager, error) {
	if dsn == ":memory:" {
		return NewMemoryPager(), nil
	}

	return nil, fmt.Errorf("WIP")
}

func NewMemoryPager() *MemoryPager {
	return &MemoryPager{
		pages:     map[PageID][]byte{},
		committed: map[PageID][]byte{},
		nextID:    1, // RootSchemaPageID (0) is reserved
	}
}
