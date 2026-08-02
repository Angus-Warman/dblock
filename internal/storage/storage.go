package storage

import (
	"errors"
	"fmt"
	"sync"
)

type BlockID int64
type BlockOffset int64
type WalID int64
type TxID int64

const BlockSize = 1024 * 8

type TxStorage struct {
	id          TxID
	wal         *WalStorage
	maxWalBlock WalID // at time of open
	dirtyBlocks []Block
}

// GetBlock returns either the committed block from main, or the WAL block that existed when TxStorage first opened
func (s *TxStorage) GetBlock(id BlockID) ([]byte, error) {
	return s.wal.GetBlock(id, s.maxWalBlock)
}

func (s *TxStorage) PutBlock(id BlockID, buf []byte) {
	idx := -1

	block := Block{
		ID:  id,
		Buf: buf,
	}

	for i, block := range s.dirtyBlocks {
		if block.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.dirtyBlocks[idx] = block
		return
	}

	s.dirtyBlocks = append(s.dirtyBlocks, block)
}

func (s *TxStorage) Commit() error {
	err := s.wal.PutBlocks(s.dirtyBlocks)

	if err != nil {
		return err
	}

	s.dirtyBlocks = []Block{}
	// Should delete itself
	return nil
}

func (s *TxStorage) Rollback() error {
	// Should delete itself
	return nil
}

type WalStorage struct {
	mu            sync.Mutex
	walIndex      map[BlockID]WalID
	walOffsets    map[WalID]BlockOffset
	txUsingWalIDs map[TxID]WalID
	main          StorageFile
	wal           StorageFile
	maxWalID      WalID
}

func (s *WalStorage) Close() error {
	// Sync?

	mainErr := s.main.Close()
	walErr := s.wal.Close()

	return errors.Join(mainErr, walErr)
}

func OpenWalStorage(dsn string) (*WalStorage, error) {
	if dsn == ":memory:" {
		return NewWalPager(
			OpenInMemoryFile("memory-main"),
			OpenInMemoryFile("memory-wal"),
		), nil
	}

	mainFp := dsn
	walFp := dsn + "-wal"

	main, err := OpenActualFile(mainFp)

	if err != nil {
		return nil, err
	}

	wal, err := OpenActualFile(walFp)

	if err != nil {
		return nil, err
	}

	return NewWalPager(main, wal), nil
}

func NewWalPager(main StorageFile, wal StorageFile) *WalStorage {
	return &WalStorage{
		walIndex:      make(map[BlockID]WalID),
		walOffsets:    make(map[WalID]BlockOffset),
		txUsingWalIDs: make(map[TxID]WalID),
		main:          main,
		wal:           wal,
		maxWalID:      0, // Or read from wal?
	}
}

func (w *WalStorage) NewTxPager() *TxStorage {
	return &TxStorage{
		wal:         w,
		maxWalBlock: w.maxWalID,
	}
}

// BlockID(8) + dbSizeAfter(8) + checksum(8).
const frameHeaderSize = 24

func pageOffset(id BlockID) (BlockOffset, BlockOffset) {
	start := id * BlockSize
	return BlockOffset(start), BlockOffset(start + BlockSize)
}

var ErrEmptyBlock = errors.New("empty block")

func (p *WalStorage) GetBlock(id BlockID, toMax WalID) ([]byte, error) {
	p.mu.Lock()
	walID, inWAL := p.walIndex[id]
	p.mu.Unlock()

	if inWAL && walID <= toMax {
		buf := make([]byte, BlockSize)
		if _, err := p.wal.ReadAt(buf, int64(walID+frameHeaderSize)); err != nil {
			return nil, fmt.Errorf("GetPage: %w", err)
		}
		return buf, nil
	}

	// Else get from main
	start, end := pageOffset(id)
	mainSize, err := p.main.Size()

	if err != nil {
		return nil, err
	}

	if int64(end) > mainSize {
		return nil, ErrEmptyBlock
	}

	buf := make([]byte, BlockSize)
	if _, err := p.main.ReadAt(buf, int64(start)); err != nil {
		return nil, fmt.Errorf("GetPage: %w", err)
	}

	// Checksum here?

	return buf, nil
}

type Block struct {
	ID  BlockID
	Buf []byte
}

// Writes to WAL
func (p *WalStorage) PutBlocks(blocks []Block) error {
	return fmt.Errorf("WIP")
}

// Copies all blocks that are not required by active TxStorage from WAL to Main
func (p *WalStorage) Checkpoint() error {
	return fmt.Errorf("WIP")
}
