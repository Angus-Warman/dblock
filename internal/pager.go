package internal

import (
	"fmt"
	"sync"
)

type TxPager struct {
	pager Pager
	dirty map[PageID]*Page
}

// Close implements [Pager].
func (p *TxPager) Close() error {
	return p.pager.Close()
}

func NewTxPager(pager Pager) *TxPager {
	return &TxPager{
		pager: pager,
		dirty: make(map[PageID]*Page),
	}
}

func (p *TxPager) Commit() error {
	for pageID, page := range p.dirty {
		err := p.pager.PutPage(pageID, page)

		if err != nil {
			return fmt.Errorf("Commit: %w", err) // TODO needs to undo saved changes
		}
	}

	p.dirty = make(map[PageID]*Page)
	return nil
}

func (p *TxPager) Rollback() error {
	p.dirty = make(map[PageID]*Page)
	return nil
}

// NextID implements [Pager].
func (p *TxPager) NextID() PageID {
	return p.pager.NextID()
}

// PutPage implements [Pager].
func (p *TxPager) PutPage(id PageID, page *Page) error {
	p.dirty[id] = page
	return nil
}

// GetPage implements [Pager].
func (p *TxPager) GetPage(id PageID) (*Page, error) {
	dirty, ok := p.dirty[id]

	if ok {
		return dirty, nil
	}

	return p.pager.GetPage(id)
}

type Pager interface {
	GetPage(PageID) (*Page, error)
	PutPage(PageID, *Page) error
	NextID() PageID
	Close() error
}

type MemoryPager struct {
	file   []byte
	nextID PageID
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
	m.file = ensurePage(m.file, id)
	start, end := pageOffset(id)
	copy(m.file[start:end], data)
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

var (
	memPagerLock   sync.Mutex
	globalMemPager *MemoryPager
	memPagerREfs   = 0
)

func NewPager(dsn string) (Pager, error) {
	if dsn == ":memory:" {
		memPagerLock.Lock()
		defer memPagerLock.Unlock()

		if globalMemPager == nil {
			p, err := NewMemoryPager()

			if err != nil {
				return nil, err
			}

			globalMemPager = p
		}

		memPagerREfs++
		return globalMemPager, nil
	}

	return NewFilePager(dsn)
}

func (p *MemoryPager) Close() error {
	memPagerLock.Lock()
	defer memPagerLock.Unlock()

	memPagerREfs--
	if memPagerREfs > 0 {
		return nil
	}

	globalMemPager = nil
	return nil
}
