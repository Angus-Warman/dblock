package internal

import (
	"dblock2/internal/metadata"
	"dblock2/internal/storage"
	"errors"
)

type Pager interface {
	GetPage(PageID) (*Page, error)
	PutPage(PageID, *Page) error
	GetMetadata() (*metadata.Metadata, error)
	PutMetadata(*metadata.Metadata) error
	NextID() PageID
	Close() error
}

type PagerSource struct {
	wal *storage.WalStorage
}

func NewPagerSource(dsn string) (*PagerSource, error) {
	wal, err := storage.OpenWalStorage(dsn)

	if err != nil {
		return nil, err
	}

	return &PagerSource{
		wal: wal,
	}, nil
}

func (s *PagerSource) Close() error {
	return s.wal.Close()
}

func (s *PagerSource) Begin() (*StoragePager, error) {
	tx := s.wal.NewTxStorage()

	p := &StoragePager{
		store:  tx,
		nextID: PageID(tx.NextBlockID()),
	}

	m, err := p.GetMetadata()

	if err != nil {
		return nil, err
	}

	p.pageSize = m.PageSize()

	return p, nil
}

type StoragePager struct {
	store    *storage.TxStorage
	nextID   PageID
	pageSize int
}

// GetPage implements [Pager].
func (p *StoragePager) GetPage(id PageID) (*Page, error) {
	buf, err := p.store.GetBlock(storage.BlockID(id))

	if err != nil {
		if errors.Is(err, storage.ErrEmptyBlock) {
			return nil, ErrEmptyPage
		}
		return nil, err
	}

	// The root schema page shares its block with the database metadata.
	if id == RootSchemaPageID {
		_, buf = splitHeaderPageData(buf)
	}

	return Decode(buf)
}

func (p *StoragePager) GetMetadata() (*metadata.Metadata, error) {
	buf, err := p.store.GetBlock(storage.BlockID(RootSchemaPageID))

	if err != nil {
		if errors.Is(err, storage.ErrEmptyBlock) {
			return metadata.New(), nil
		}
		return nil, err
	}

	buf, _ = splitHeaderPageData(buf)

	return metadata.Decode(buf)
}

func (p *StoragePager) PutMetadata(m *metadata.Metadata) error {
	m.NumberOfPages = uint32(p.store.NextBlockID())
	m.CalculateChecksum()

	blockSize := m.PageSize()

	buf, err := p.store.GetBlock(storage.BlockID(RootSchemaPageID))

	if err != nil {
		if !errors.Is(err, storage.ErrEmptyBlock) {
			return err
		}

		// No root page yet: write the metadata ahead of an empty page region.
		buf = make([]byte, blockSize-metadata.Length)
	} else {
		_, buf = splitHeaderPageData(buf)
	}

	p.store.PutBlock(storage.BlockID(RootSchemaPageID), joinHeaderPageData(m, buf))
	p.pageSize = blockSize
	return nil
}

// PutPage implements [Pager].
func (p *StoragePager) PutPage(id PageID, page *Page) error {
	buf, err := page.Encode(p.pageSize)

	if err != nil {
		return err
	}

	// The root schema page shares its block with the database metadata, which
	// must survive the page being rewritten.
	if id == RootSchemaPageID {
		m, err := p.GetMetadata()

		if err != nil {
			return err
		}

		buf = joinHeaderPageData(m, buf)
	}

	p.store.PutBlock(storage.BlockID(id), buf)
	return nil
}

// NextID implements [Pager].
func (p *StoragePager) NextID() PageID {
	id := p.nextID
	p.nextID++
	p.store.ReserveBlock(storage.BlockID(id))
	return id
}

func (p *StoragePager) Commit() error {
	return p.store.Commit()
}

func (p *StoragePager) Rollback() error {
	return p.store.Rollback()
}

// Close implements [Pager].
func (p *StoragePager) Close() error {
	return p.store.Close()
}
