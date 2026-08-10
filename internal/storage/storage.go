package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"

	"github.com/Angus-Warman/dblock/internal/metadata"
)

// The first block of a database file reserves metadata.Length bytes for
// database metadata; its file-change counter is incremented every time pages
// move from the WAL into main.

type BlockID int64
type BlockOffset int64
type WalID int64
type TxID int64

const BlockSize = 1024 * 8 // default block size (2^13), overridden per-instance from metadata

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

	if s.wal.ShouldCheckpoint() {
		if err := s.wal.Checkpoint(); err != nil {
			return err
		}
	}

	return nil
}

func (s *TxStorage) Rollback() error {
	s.dirtyBlocks = make(map[BlockID]Block)
	s.wal.forgetTx(s.id)
	return nil
}

// Close closes the underlying WAL storage.
func (s *TxStorage) Close() error {
	return s.wal.Close()
}

// NextBlockID returns the lowest block ID not used by any committed block,
// for handing out new page IDs.
func (s *TxStorage) NextBlockID() BlockID {
	return s.wal.NextBlockID()
}

// ReserveBlock records that the given block ID is in use.
func (s *TxStorage) ReserveBlock(id BlockID) {
	s.wal.ReserveBlock(id)
}

type WalStorage struct {
	mu               sync.Mutex
	walIndex         map[BlockID]WalID
	walOffsets       map[WalID]BlockOffset
	txUsingWalIDs    map[TxID]WalID
	main             File
	wal              File
	maxWalID         WalID
	nextTxID         TxID
	walEnd           BlockOffset
	nextBlockID      BlockID
	committedTxCount int
	blockSize        int
}

// Checkpointing policy: a checkpoint is triggered once 100 transactions have
// committed since the last checkpoint, or as soon as the WAL exceeds 100
// pages (frames).
const (
	checkpointTxInterval   = 100
	checkpointWalPageLimit = 100
)

func (s *WalStorage) Close() error {
	checkpointErr := s.Checkpoint()
	mainErr := s.main.Close()
	walErr := s.wal.Close()

	return errors.Join(checkpointErr, mainErr, walErr)
}

func OpenWalStorage(dsn string) (*WalStorage, error) {
	if dsn == ":memory:" {
		return NewWalStorage(
			OpenMemoryFile("memory-main"),
			OpenMemoryFile("memory-wal"),
		)
	}

	mainFp := dsn
	walFp := dsn + "-wal"

	main, err := OpenDiskFile(mainFp)

	if err != nil {
		return nil, err
	}

	wal, err := OpenDiskFile(walFp)

	if err != nil {
		return nil, err
	}

	return NewWalStorage(main, wal)
}

func NewWalStorage(main File, wal File) (*WalStorage, error) {
	// The page size is fixed once the first page is written, so the block size
	// can be read straight out of the metadata header at the start of main.
	blockSize, err := detectBlockSize(main)
	if err != nil {
		return nil, err
	}

	// Reopened files may already hold pages in main; never allocate below them.
	nextBlockID := BlockID(1)

	size, err := main.Size()

	if err != nil {
		return nil, err
	}

	if n := BlockID(size / int64(blockSize)); n > nextBlockID {
		nextBlockID = n
	}

	s := &WalStorage{
		walIndex:      make(map[BlockID]WalID),
		walOffsets:    make(map[WalID]BlockOffset),
		txUsingWalIDs: make(map[TxID]WalID),
		main:          main,
		wal:           wal,
		maxWalID:      0,
		nextBlockID:   nextBlockID,
		blockSize:     blockSize,
	}

	if err := s.Recover(); err != nil {
		return nil, fmt.Errorf("NewWalStorage: %w", err)
	}

	// Recovered WAL frames may name blocks past main's current size; never
	// allocate below them either.
	for id := range s.walIndex {
		if id >= s.nextBlockID {
			s.nextBlockID = id + 1
		}
	}

	return s, nil
}

// Recover replays the WAL from the start, validating each frame's checksum,
// and rebuilds the in-memory index from every frame up to (and including) the
// last complete commit. Anything after that -- a torn frame, a bad checksum,
// or a dangling uncommitted frame -- is discarded, so a transaction is only
// visible once its commit marker has been fully written.
func (p *WalStorage) Recover() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.walIndex = make(map[BlockID]WalID)
	p.walOffsets = make(map[WalID]BlockOffset)
	p.maxWalID = 0

	size, err := p.wal.Size()

	if err != nil {
		return err
	}

	header := make([]byte, frameHeaderSize)
	buf := make([]byte, p.blockSize)
	pendingIndex := make(map[BlockID]WalID)
	pendingOffsets := make(map[WalID]BlockOffset)
	var offset BlockOffset
	var lastCommitEnd BlockOffset

	for int64(offset)+frameHeaderSize <= size {
		if _, err := p.wal.ReadAt(header, int64(offset)); err != nil {
			break // torn header, stop here
		}

		blockID := BlockID(binary.BigEndian.Uint64(header[0:8]))
		commitMarker := binary.BigEndian.Uint64(header[8:16])
		storedChecksum := uint32(binary.BigEndian.Uint64(header[16:24]))

		if int64(offset)+int64(p.frameSize()) > size {
			break // torn payload, stop here
		}
		if _, err := p.wal.ReadAt(buf, int64(offset+frameHeaderSize)); err != nil {
			break
		}

		if crc32.ChecksumIEEE(buf) != storedChecksum {
			break
		}

		walID := p.maxWalID
		p.maxWalID++
		pendingIndex[blockID] = walID
		pendingOffsets[walID] = offset
		offset += BlockOffset(p.frameSize())

		if commitMarker != 0 {
			// The transaction is committed: everything accumulated since the
			// last commit marker is now visible.
			for id, walID := range pendingIndex {
				p.walIndex[id] = walID
			}
			for walID, off := range pendingOffsets {
				p.walOffsets[walID] = off
			}
			lastCommitEnd = offset
		}
	}

	p.walEnd = lastCommitEnd
	return nil
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

// ReserveBlock records that the given block ID is in use so fresh allocations
// never reuse it, even before the block is first written.
func (w *WalStorage) ReserveBlock(id BlockID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if id >= w.nextBlockID {
		w.nextBlockID = id + 1
	}
}

// NextBlockID returns the lowest block ID not yet allocated, for handing out
// new page IDs.
func (w *WalStorage) NextBlockID() BlockID {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.nextBlockID
}

// detectBlockSize reads the page size from main's metadata header, falling back
// to the default for a database whose first page has not been written yet.
func detectBlockSize(main File) (int, error) {
	size, err := main.Size()

	if err != nil {
		return 0, err
	}

	if size < metadata.Length {
		return BlockSize, nil
	}

	buf := make([]byte, metadata.Length)

	if _, err := main.ReadAt(buf, 0); err != nil {
		return 0, err
	}

	m, err := metadata.Decode(buf)

	if err != nil {
		return 0, err
	}

	return m.PageSize(), nil
}

// BlockID(8) + dbSizeAfter(8) + checksum(8).
const frameHeaderSize = 24

func (p *WalStorage) frameSize() int {
	return frameHeaderSize + p.blockSize
}

func (p *WalStorage) pageOffset(id BlockID) (BlockOffset, BlockOffset) {
	start := BlockOffset(id) * BlockOffset(p.blockSize)
	return start, start + BlockOffset(p.blockSize)
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
		buf := make([]byte, p.blockSize)
		if _, err := p.wal.ReadAt(buf, int64(p.walOffsets[walID]+frameHeaderSize)); err != nil {
			return nil, fmt.Errorf("GetPage: %w", err)
		}
		return buf, nil
	}

	// Else get from main
	start, end := p.pageOffset(id)
	mainSize, err := p.main.Size()

	if err != nil {
		return nil, err
	}

	if int64(end) > mainSize {
		return nil, ErrEmptyBlock
	}

	buf := make([]byte, p.blockSize)
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

	// A brand-new database may configure its page size before the first block
	// is ever written; adopt the resulting block size for every frame.
	if p.nextBlockID <= 1 && len(blocks[0].Buf) != p.blockSize {
		p.blockSize = len(blocks[0].Buf)
	}

	for _, block := range blocks {
		if len(block.Buf) != p.blockSize {
			return fmt.Errorf("PutBlocks: block %d has size %d, want %d", block.ID, len(block.Buf), p.blockSize)
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
		if block.ID >= p.nextBlockID {
			p.nextBlockID = block.ID + 1
		}
		offset += BlockOffset(p.frameSize())
	}

	if err := p.wal.Sync(); err != nil {
		return fmt.Errorf("PutBlocks: %w", err)
	}

	p.walEnd = offset
	p.committedTxCount++
	return nil
}

// GetMetadata returns the committed database metadata stored in the header of
// block 0, or a fresh zeroed metadata when block 0 does not exist yet.
func (p *WalStorage) GetMetadata() (*metadata.Metadata, error) {
	buf, err := p.GetBlock(BlockID(0), p.maxWalID)

	if err != nil {
		if errors.Is(err, ErrEmptyBlock) {
			return metadata.New(), nil
		}
		return nil, err
	}

	return metadata.Decode(buf[:metadata.Length])
}

// PutMetadata overwrites block 0's metadata header, preserving the page data
// that follows it.
func (p *WalStorage) PutMetadata(m *metadata.Metadata) error {
	buf, err := p.GetBlock(BlockID(0), p.maxWalID)

	if err != nil {
		if !errors.Is(err, ErrEmptyBlock) {
			return err
		}

		buf = make([]byte, p.blockSize)
	}

	full := make([]byte, 0, p.blockSize)
	full = append(full, m.Encode()...)
	full = append(full, buf[metadata.Length:]...)

	return p.PutBlocks([]Block{{ID: BlockID(0), Buf: full}})
}

func (p *WalStorage) ShouldCheckpoint() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.committedTxCount > checkpointTxInterval {
		return true
	}

	numWalPages := p.walEnd / BlockOffset(p.frameSize())

	return numWalPages > checkpointWalPageLimit
}

// Checkpoint copies committed WAL frames that no active TxStorage depends on
// into main, fsyncs main, then rewrites the WAL keeping only the frames that
// active transactions still need for their snapshot.
func (p *WalStorage) Checkpoint() error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	// If no committed frame can safely move to main, checkpointing is a no-op.
	if !pagesMoveToMain(p.walIndex, gate) {
		return nil
	}

	// 1. Copy every safe frame into main.
	for id, walID := range p.walIndex {
		if walID >= gate {
			continue // an active TxStorage still needs this version
		}
		buf := make([]byte, p.blockSize)
		if _, err := p.wal.ReadAt(buf, int64(p.walOffsets[walID]+frameHeaderSize)); err != nil {
			return fmt.Errorf("Checkpoint: %w", err)
		}
		start, _ := p.pageOffset(id)
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
		buf := make([]byte, p.frameSize())
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
		offset += BlockOffset(p.frameSize())
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

	if err := p.main.Sync(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}
	if err := p.wal.Sync(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	if err := p.bumpFileChangeCounter(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	// Sync again after file increment
	if err := p.main.Sync(); err != nil {
		return fmt.Errorf("Checkpoint: %w", err)
	}

	p.committedTxCount = 0
	return nil
}

// pagesMoveToMain reports whether any committed frame will be copied from the
// WAL into main during this checkpoint.
func pagesMoveToMain(walIndex map[BlockID]WalID, gate WalID) bool {
	for _, walID := range walIndex {
		if walID < gate {
			return true
		}
	}
	return false
}

func (p *WalStorage) getMetadataHot() (*metadata.Metadata, error) {
	buf := make([]byte, metadata.Length)

	if _, err := p.main.ReadAt(buf, 0); err != nil {
		return nil, err
	}

	return metadata.Decode(buf)
}

func (p *WalStorage) putMetadataHot(m *metadata.Metadata) error {
	_, err := p.main.WriteAt(m.Encode(), 0)
	return err
}

func (p *WalStorage) bumpFileChangeCounter() error {
	m, err := p.getMetadataHot()

	if err != nil {
		return err
	}

	m.FileVersion++

	return p.putMetadataHot(m)
}
