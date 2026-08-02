package internal

import (
	"dblock2/internal/storage"
	"errors"
)

type Pager interface {
	GetPage(PageID) (*Page, error)
	PutPage(PageID, *Page) error
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

func (s *PagerSource) Begin() *StoragePager {
	tx := s.wal.NewTxStorage()
	return &StoragePager{
		store:  tx,
		nextID: PageID(tx.NextBlockID()),
	}
}

type StoragePager struct {
	store  *storage.TxStorage
	nextID PageID
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

	return Decode(buf)
}

// PutPage implements [Pager].
func (p *StoragePager) PutPage(id PageID, page *Page) error {
	buf, err := page.Encode()

	if err != nil {
		return err
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
