package storage

import (
	"errors"
	"fmt"
	"sync"
)

type BlockID int64
type BlockOffset int64

const BlockSize = 1024 * 8

type TxStorage struct {
	wal         *WalStorage
	maxPage     BlockID // at time of open
	dirtyBlocks []Block
}

func (s *TxStorage) GetBlock(id BlockID) ([]byte, error) {
	return s.wal.GetBlock(id) // To max?
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
	return nil
}

type WalStorage struct {
	mu       sync.Mutex
	walIndex map[BlockID]BlockOffset
	main     StorageFile
	wal      StorageFile
	maxBlock BlockID
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
		main:     main,
		wal:      wal,
		maxBlock: 0, // Or read from wal?
	}
}

func (w *WalStorage) NewTxPager() *TxStorage {
	return &TxStorage{
		wal:     w,
		maxPage: w.maxBlock,
	}
}

// BlockID(8) + dbSizeAfter(8) + checksum(8).
const frameHeaderSize = 24

func pageOffset(id BlockID) (BlockOffset, BlockOffset) {
	start := id * BlockSize
	return BlockOffset(start), BlockOffset(start + BlockSize)
}

var ErrEmptyBlock = errors.New("empty block")

func (p *WalStorage) GetBlock(id BlockID) ([]byte, error) {
	p.mu.Lock()
	offset, inWAL := p.walIndex[id]
	p.mu.Unlock()

	if inWAL {
		buf := make([]byte, BlockSize)
		if _, err := p.wal.ReadAt(buf, int64(offset+frameHeaderSize)); err != nil {
			return nil, fmt.Errorf("GetPage: %w", err)
		}
		return buf, nil
	}

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
