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
	wal     *WalStorage
	maxPage BlockID // at time of open
}

type WalStorage struct {
	mu       sync.Mutex
	walIndex map[BlockID]BlockOffset
	main     StorageFile
	wal      StorageFile
	maxBlock BlockID
}

func OpenWalPager(dsn string) (*WalStorage, error) {
	if dsn == ":memory:" {
		return NewWalPager(OpenInMemoryFile("memory-main"), OpenInMemoryFile("memory-wal")), nil
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

// import (
// 	"encoding/binary"
// 	"fmt"
// 	"hash/crc32"
// 	"sync"
// )

// // frameHeaderSize is pageID(4) + dbSizeAfter(4) + checksum(4).
// const frameHeaderSize = 12

// type WALPager struct {

// 	main StorageFile
// 	wal  StorageFile

// 	// index maps a page to the offset of its most recent frame in the WAL.
// 	// Pages not present here live only in "main".
// 	index map[PageID]int64

// 	// walEnd is the byte offset one past the last valid (fully committed)
// 	// frame in the WAL file. Anything in the file past this point is
// 	// either torn or not yet written.
// 	walEnd int64

// 	// readerCount tracks active readers so Close()/Checkpoint() know
// 	// whether it's safe to fully reclaim the WAL. See note above: since
// 	// readers don't hold snapshots yet, this is a simple busy-count, not
// 	// per-reader watermarks. Extending to real snapshots means replacing
// 	// this with a slice of watermark offsets and gating on their min.
// 	readerCount int

// 	writerMu sync.Mutex // only one writer transaction at a time

// 	nextID PageID
// }

// // NewWALPager wires up a WALPager over the given main/WAL RawFiles and
// // replays the WAL to rebuild the in-memory index (crash recovery).
// func NewWALPager(main, wal StorageFile) (*WALPager, error) {
// 	p := &WALPager{
// 		main:  main,
// 		wal:   wal,
// 		index: make(map[PageID]int64),
// 	}

// 	if err := p.recover(); err != nil {
// 		return nil, fmt.Errorf("NewWALPager: %w", err)
// 	}

// 	if err := p.initNextID(); err != nil {
// 		return nil, fmt.Errorf("NewWALPager: %w", err)
// 	}

// 	return p, nil
// }

// // recover scans the WAL from the start, validating checksums, and
// // rebuilds the index from every frame up to (and including) the last
// // complete commit. Anything after that -- a torn frame, a bad checksum,
// // or a dangling uncommitted frame -- is discarded.
// func (p *WALPager) recover() error {
// 	size, err := p.wal.Size()
// 	if err != nil {
// 		return err
// 	}

// 	var offset int64
// 	var lastCommitEnd int64 // walEnd as of the last valid commit marker
// 	pendingIndex := make(map[PageID]int64)

// 	for offset+frameHeaderSize <= size {
// 		header := make([]byte, frameHeaderSize)
// 		if _, err := p.wal.ReadAt(header, offset); err != nil {
// 			break // torn header, stop here
// 		}

// 		pageID := PageID(binary.BigEndian.Uint32(header[0:4]))
// 		dbSizeAfter := binary.BigEndian.Uint32(header[4:8])
// 		storedChecksum := binary.BigEndian.Uint32(header[8:12])

// 		pageData := make([]byte, PageSize)
// 		if offset+int64(frameHeaderSize)+int64(PageSize) > size {
// 			break // torn page payload, stop here
// 		}
// 		if _, err := p.wal.ReadAt(pageData, offset+frameHeaderSize); err != nil {
// 			break
// 		}

// 		if crc32.ChecksumIEEE(pageData) != storedChecksum {
// 			break
// 		}

// 		frameOffset := offset
// 		pendingIndex[pageID] = frameOffset
// 		offset += frameHeaderSize + int64(PageSize)

// 		if dbSizeAfter != 0 {
// 			// Commit marker: everything accumulated in pendingIndex so far
// 			// (across possibly multiple frames/transactions) is now valid.
// 			for id, off := range pendingIndex {
// 				p.index[id] = off
// 			}
// 			lastCommitEnd = offset
// 		}
// 	}

// 	p.walEnd = lastCommitEnd
// 	return nil
// }

// // initNextID scans main + WAL-indexed pages to find the next free PageID.
// // A real implementation would likely persist this in a header page instead
// // of recomputing it, but this is enough to keep NextID correct after
// // recovery.
// func (p *WALPager) initNextID() error {
// 	var max PageID
// 	seen := false

// 	for id := range p.index {
// 		if !seen || id > max {
// 			max = id
// 			seen = true
// 		}
// 	}

// 	mainSize, err := p.main.Size()
// 	if err != nil {
// 		return err
// 	}
// 	if mainPages := PageID(mainSize / PageSize); mainPages > 0 {
// 		if !seen || mainPages-1 > max {
// 			max = mainPages - 1
// 			seen = true
// 		}
// 	}

// 	if seen {
// 		p.nextID = max + 1
// 	}
// 	return nil
// }

// // NextID implements [Pager].
// func (p *WALPager) NextID() PageID {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	id := p.nextID
// 	p.nextID++
// 	return id
// }

const frameHeaderSize = 12

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

	return buf, nil
}

// // PutPage implements [Pager]. Callers are expected to go through
// // TxPager for buffering + Commit(); WALPager.PutPage writes (and commits)
// // a single page immediately as its own transaction.
// func (p *WALPager) PutPage(id PageID, page *Page) error {
// 	return p.CommitPages(map[PageID]*Page{id: page})
// }

// // CommitPages writes a batch of pages to the WAL as a single atomic
// // transaction: every frame is appended, and only the last frame carries
// // the commit marker (dbSizeAfter != 0), so a crash mid-write leaves no
// // partially-visible transaction behind.
// func (p *WALPager) CommitPages(pages map[PageID]*Page) error {
// 	if len(pages) == 0 {
// 		return nil
// 	}

// 	p.writerMu.Lock()
// 	defer p.writerMu.Unlock()

// 	p.mu.Lock()
// 	writeOffset := p.walEnd
// 	dbSizeAfter := uint32(p.nextID) // simple monotonic marker; just needs to be non-zero
// 	p.mu.Unlock()

// 	// Stable order isn't required for correctness, only that the last
// 	// frame written is the one carrying the commit marker.
// 	ids := make([]PageID, 0, len(pages))
// 	for id := range pages {
// 		ids = append(ids, id)
// 	}

// 	newIndex := make(map[PageID]int64, len(ids))
// 	offset := writeOffset

// 	for i, id := range ids {
// 		data, err := pages[id].Encode()
// 		if err != nil {
// 			return fmt.Errorf("CommitPages: %w", err)
// 		}
// 		if len(data) != PageSize {
// 			return fmt.Errorf("CommitPages: encoded page %d has size %d, want %d", id, len(data), PageSize)
// 		}

// 		header := make([]byte, frameHeaderSize)
// 		binary.BigEndian.PutUint32(header[0:4], uint32(id))
// 		if i == len(ids)-1 {
// 			binary.BigEndian.PutUint32(header[4:8], dbSizeAfter)
// 		}
// 		binary.BigEndian.PutUint32(header[8:12], crc32.ChecksumIEEE(data))

// 		if _, err := p.wal.WriteAt(header, offset); err != nil {
// 			return fmt.Errorf("CommitPages: %w", err)
// 		}
// 		if _, err := p.wal.WriteAt(data, offset+frameHeaderSize); err != nil {
// 			return fmt.Errorf("CommitPages: %w", err)
// 		}

// 		newIndex[id] = offset
// 		offset += frameHeaderSize + int64(PageSize)
// 	}

// 	if err := p.wal.Sync(); err != nil {
// 		return fmt.Errorf("CommitPages: %w", err)
// 	}

// 	p.mu.Lock()
// 	for id, off := range newIndex {
// 		p.index[id] = off
// 	}
// 	p.walEnd = offset
// 	p.mu.Unlock()

// 	return nil
// }

// // BeginRead / EndRead mark a reader as active so Checkpoint/Close know
// // it's not safe to fully reclaim the WAL. As noted on WALPager, this is a
// // busy-count rather than a true per-reader snapshot watermark.
// func (p *WALPager) BeginRead() {
// 	p.mu.Lock()
// 	p.readerCount++
// 	p.mu.Unlock()
// }

// func (p *WALPager) EndRead() {
// 	p.mu.Lock()
// 	if p.readerCount > 0 {
// 		p.readerCount--
// 	}
// 	p.mu.Unlock()
// }

// // Checkpoint copies committed WAL frames into the main DB file and
// // truncates the WAL, following the crash-safe order:
// //
// //  1. write every page's latest version into main
// //  2. fsync main
// //  3. only then truncate/reset the WAL
// //
// // If a crash happens before step 2 completes, the WAL is untouched and
// // recovery just replays it as usual. If it happens between step 2 and 3,
// // replaying the (not-yet-truncated) WAL again on restart is harmless,
// // since checkpointing is idempotent.
// //
// // Checkpoint is a no-op (returns nil, false) if there are active readers,
// // since it isn't safe to reclaim WAL frames they might still need.
// func (p *WALPager) Checkpoint() (ran bool, err error) {
// 	p.writerMu.Lock()
// 	defer p.writerMu.Unlock()

// 	p.mu.Lock()
// 	if p.readerCount > 0 {
// 		p.mu.Unlock()
// 		return false, nil
// 	}
// 	// Snapshot the index/walEnd under lock, then release -- no writer can
// 	// run concurrently anyway because we hold writerMu.
// 	indexCopy := make(map[PageID]int64, len(p.index))
// 	for id, off := range p.index {
// 		indexCopy[id] = off
// 	}
// 	p.mu.Unlock()

// 	if len(indexCopy) == 0 {
// 		return true, nil
// 	}

// 	// 1. Write every page's latest version into main.
// 	for id, walOffset := range indexCopy {
// 		buf := make([]byte, PageSize)
// 		if _, err := p.wal.ReadAt(buf, walOffset+frameHeaderSize); err != nil {
// 			return false, fmt.Errorf("Checkpoint: %w", err)
// 		}
// 		start, _ := pageOffset(id)
// 		if _, err := p.main.WriteAt(buf, int64(start)); err != nil {
// 			return false, fmt.Errorf("Checkpoint: %w", err)
// 		}
// 	}

// 	// 2. fsync main -- every page above must be durable before we touch
// 	// the WAL at all.
// 	if err := p.main.Sync(); err != nil {
// 		return false, fmt.Errorf("Checkpoint: %w", err)
// 	}

// 	// 3. Only now is it safe to reclaim the WAL.
// 	if err := p.wal.Truncate(0); err != nil {
// 		return false, fmt.Errorf("Checkpoint: %w", err)
// 	}
// 	if err := p.wal.Sync(); err != nil {
// 		return false, fmt.Errorf("Checkpoint: %w", err)
// 	}

// 	p.mu.Lock()
// 	p.index = make(map[PageID]int64)
// 	p.walEnd = 0
// 	p.mu.Unlock()

// 	return true, nil
// }

// // Close attempts a best-effort checkpoint, then closes both underlying
// // files. A checkpoint at close is purely an optimization (smaller WAL,
// // faster next open) -- correctness never depends on it succeeding, so a
// // failed or skipped checkpoint here does not make Close itself fail.
// func (p *WALPager) Close() error {
// 	if _, err := p.Checkpoint(); err != nil {
// 		// Leave the WAL as-is; next open will recover from it normally.
// 		_ = err
// 	}

// 	if err := p.wal.Close(); err != nil {
// 		return fmt.Errorf("Close: %w", err)
// 	}
// 	if err := p.main.Close(); err != nil {
// 		return fmt.Errorf("Close: %w", err)
// 	}
// 	return nil
// }

// // Close implements [Pager].
// func (p *TxPager) Close() error {
// 	return p.pager.Close()
// }

// // Commit flushes all buffered pages as a single atomic WAL transaction
// // when the underlying pager supports it (WALPager), falling back to
// // per-page PutPage otherwise.
// func (p *TxPager) Commit() error {
// 	if len(p.dirty) == 0 {
// 		return nil
// 	}

// 	if batch, ok := p.pager.(interface {
// 		CommitPages(map[PageID]*Page) error
// 	}); ok {
// 		if err := batch.CommitPages(p.dirty); err != nil {
// 			return fmt.Errorf("Commit: %w", err)
// 		}
// 		p.dirty = make(map[PageID]*Page)
// 		return nil
// 	}

// 	for pageID, page := range p.dirty {
// 		if err := p.pager.PutPage(pageID, page); err != nil {
// 			return fmt.Errorf("Commit: %w", err) // TODO needs to undo saved changes
// 		}
// 	}

// 	p.dirty = make(map[PageID]*Page)
// 	return nil
// }

// func (p *TxPager) Rollback() error {
// 	p.dirty = make(map[PageID]*Page)
// 	return nil
// }

// // NextID implements [Pager].
// func (p *TxPager) NextID() PageID {
// 	return p.pager.NextID()
// }

// // PutPage implements [Pager].
// func (p *TxPager) PutPage(id PageID, page *Page) error {
// 	p.dirty[id] = page
// 	return nil
// }

// // GetPage implements [Pager].
// func (p *TxPager) GetPage(id PageID) (*Page, error) {
// 	if dirty, ok := p.dirty[id]; ok {
// 		return dirty, nil
// 	}
// 	return p.pager.GetPage(id)
// }

// // pageOffset returns the byte range of the page with the given ID.
// func pageOffset(id PageID) (int, int) {
// 	start := int(id) * PageSize
// 	return start, start + PageSize
// }

// func NewPager(dsn string) (Pager, error) {
// 	if dsn == ":memory:" {
// 		memPagerLock.Lock()
// 		defer memPagerLock.Unlock()

// 		if globalMemPager == nil {
// 			p, err := NewWALPager(OpenInMemoryFile(), OpenInMemoryFile())
// 			if err != nil {
// 				return nil, err
// 			}
// 			globalMemPager = p
// 		}

// 		memPagerRefs++
// 		return &refCountedMemPager{WALPager: globalMemPager}, nil
// 	}

// 	mainFile, err := OpenActualFile(dsn)
// 	if err != nil {
// 		return nil, fmt.Errorf("NewPager: %w", err)
// 	}
// 	walFile, err := OpenActualFile(dsn + "-wal")
// 	if err != nil {
// 		return nil, fmt.Errorf("NewPager: %w", err)
// 	}

// 	return NewWALPager(mainFile, walFile)
// }

// // refCountedMemPager wraps the shared globalMemPager so each connection's
// // Close() only tears it down once nobody else is using it, without
// // touching WALPager's own (unrelated) internal locking.
// type refCountedMemPager struct {
// 	*WALPager
// }

// func (p *refCountedMemPager) Close() error {
// 	memPagerLock.Lock()
// 	defer memPagerLock.Unlock()

// 	memPagerRefs--
// 	if memPagerRefs > 0 {
// 		return nil
// 	}

// 	err := p.WALPager.Close()
// 	globalMemPager = nil
// 	return err
// }
