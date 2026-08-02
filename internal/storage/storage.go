package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
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
	dirtyBlocks map[BlockID]Block
}

// GetBlock returns either the committed block from main, or the WAL block that existed when TxStorage first opened
func (s *TxStorage) GetBlock(id BlockID) ([]byte, error) {
	exists, ok := s.dirtyBlocks[id]

	if ok {
		return exists.Buf, nil
	}

	return s.wal.GetBlock(id, s.maxWalBlock)
}

func (s *TxStorage) PutBlock(id BlockID, buf []byte) {
	block := Block{
		ID:  id,
		Buf: buf,
	}

	s.dirtyBlocks[id] = block
}

func (s *TxStorage) Commit() error {
	blocks := []Block{}

	for _, block := range s.dirtyBlocks {
		blocks = append(blocks, block)
	}

	err := s.wal.PutBlocks(blocks)

	if err != nil {
		return err
	}

	s.dirtyBlocks = make(map[BlockID]Block)
	s.wal.forgetTx(s.id)
	return nil
}

func (s *TxStorage) Rollback() error {
	s.dirtyBlocks = make(map[BlockID]Block)
	s.wal.forgetTx(s.id)
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
	nextTxID      TxID
	walEnd        BlockOffset
}

func (s *WalStorage) Close() error {
	// Sync?

	mainErr := s.main.Close()
	walErr := s.wal.Close()

	return errors.Join(mainErr, walErr)
}

func OpenWalStorage(dsn string) (*WalStorage, error) {
	if dsn == ":memory:" {
		return NewWalStorage(
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

	return NewWalStorage(main, wal), nil
}

func NewWalStorage(main StorageFile, wal StorageFile) *WalStorage {
	return &WalStorage{
		walIndex:      make(map[BlockID]WalID),
		walOffsets:    make(map[WalID]BlockOffset),
		txUsingWalIDs: make(map[TxID]WalID),
		main:          main,
		wal:           wal,
		maxWalID:      0, // Or read from wal?
	}
}

func (w *WalStorage) NewTxStorage() *TxStorage {
	w.mu.Lock()
	defer w.mu.Unlock()

	tx := &TxStorage{
		id:          w.nextTxID,
		wal:         w,
		maxWalBlock: w.maxWalID,
		dirtyBlocks: make(map[BlockID]Block),
	}
	w.nextTxID++
	w.txUsingWalIDs[tx.id] = tx.maxWalBlock

	return tx
}

func (w *WalStorage) forgetTx(id TxID) {
	w.mu.Lock()
	delete(w.txUsingWalIDs, id)
	w.mu.Unlock()
}

// BlockID(8) + dbSizeAfter(8) + checksum(8).
const frameHeaderSize = 24

const frameSize = frameHeaderSize + BlockSize

func pageOffset(id BlockID) (BlockOffset, BlockOffset) {
	start := id * BlockSize
	return BlockOffset(start), BlockOffset(start + BlockSize)
}

var ErrEmptyBlock = errors.New("empty block")

// GetBlock reads the block with the given id as of the snapshot watermark
// toMax: the block's committed WAL version is visible only if its WalID is
// strictly less than toMax, otherwise the pre-transaction version in main wins.
func (p *WalStorage) GetBlock(id BlockID, toMax WalID) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	walID, inWAL := p.walIndex[id]

	if inWAL && walID < toMax {
		buf := make([]byte, BlockSize)
		if _, err := p.wal.ReadAt(buf, int64(p.walOffsets[walID]+frameHeaderSize)); err != nil {
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

// PutBlocks appends the given blocks to the WAL as a single atomic commit and
// publishes them to the in-memory index. Only the last frame carries a commit
// marker, so a future recovery pass can discard a torn commit.
func (p *WalStorage) PutBlocks(blocks []Block) error {
	if len(blocks) == 0 {
		return nil
	}

	for _, block := range blocks {
		if len(block.Buf) != BlockSize {
			return fmt.Errorf("PutBlocks: block %d has size %d, want %d", block.ID, len(block.Buf), BlockSize)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	header := make([]byte, frameHeaderSize)
	offset := p.walEnd

	for i, block := range blocks {
		walID := p.maxWalID
		p.maxWalID++

		// Header: BlockID(8) + dbSizeAfter(8) + checksum(8).
		binary.BigEndian.PutUint64(header[0:8], uint64(block.ID))
		if i == len(blocks)-1 {
			binary.BigEndian.PutUint64(header[8:16], 1) // commit marker
		}
		binary.BigEndian.PutUint64(header[16:24], uint64(crc32.ChecksumIEEE(block.Buf)))

		if _, err := p.wal.WriteAt(header, int64(offset)); err != nil {
			return fmt.Errorf("PutBlocks: %w", err)
		}
		if _, err := p.wal.WriteAt(block.Buf, int64(offset+frameHeaderSize)); err != nil {
			return fmt.Errorf("PutBlocks: %w", err)
		}

		p.walIndex[block.ID] = walID
		p.walOffsets[walID] = offset
		offset += frameSize
	}

	if err := p.wal.Sync(); err != nil {
		return fmt.Errorf("PutBlocks: %w", err)
	}

	p.walEnd = offset
	return nil
}

// Checkpoint copies committed WAL frames that no active TxStorage depends on
// into main, fsyncs main, then rewrites the WAL keeping only the frames that
// active transactions still need for their snapshot.
func (p *WalStorage) Checkpoint() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.walIndex) == 0 {
		return nil
	}

	// A TxStorage with snapshot watermark w reads block b from the WAL when
	// walIndex[b] < w and from main otherwise. Copying b's frame into main only
	// preserves those reads when every active TxStorage has w > walIndex[b], so
	// the smallest watermark gates how far checkpointing can go. With no active
	// transactions the gate is +inf and everything can be reclaimed.
	gate := p.maxWalID
	for _, w := range p.txUsingWalIDs {
		if w < gate {
			gate = w
		}
	}

	// 1. Copy every safe frame into main.
	for id, walID := range p.walIndex {
		if walID >= gate {
			continue // an active TxStorage still needs this version
		}
		buf := make([]byte, BlockSize)
		if _, err := p.wal.ReadAt(buf, int64(p.walOffsets[walID]+frameHeaderSize)); err != nil {
			return fmt.Errorf("Checkpoint: %w", err)
		}
		start, _ := pageOffset(id)
		if _, err := p.main.WriteAt(buf, int64(start)); err != nil {
			return fmt.Errorf("Checkpoint: %w", err)
		}
	}

	// 2. main must be durable before the WAL is touched at all.
	if err := p.main.Sync(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	// 3. Rewrite the WAL with only the frames that are still required, buffered
	// first so the rewrite never clobbers a frame that hasn't been read yet.
	type keptFrame struct {
		id    BlockID
		walID WalID
		buf   []byte
	}

	var kept []keptFrame
	for id, walID := range p.walIndex {
		if walID < gate {
			continue
		}
		buf := make([]byte, frameSize)
		if _, err := p.wal.ReadAt(buf, int64(p.walOffsets[walID])); err != nil {
			return fmt.Errorf("Checkpoint: %w", err)
		}
		kept = append(kept, keptFrame{id: id, walID: walID, buf: buf})
	}

	newIndex := make(map[BlockID]WalID, len(kept))
	newOffsets := make(map[WalID]BlockOffset, len(kept))
	offset := BlockOffset(0)

	for _, kf := range kept {
		if _, err := p.wal.WriteAt(kf.buf, int64(offset)); err != nil {
			return fmt.Errorf("Checkpoint: %w", err)
		}
		newIndex[kf.id] = kf.walID
		newOffsets[kf.walID] = offset
		offset += frameSize
	}

	if err := p.wal.Truncate(int64(offset)); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}
	if err := p.wal.Sync(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	p.walIndex = newIndex
	p.walOffsets = newOffsets
	p.walEnd = offset
	return nil
}
