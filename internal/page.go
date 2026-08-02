package internal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var LE = binary.LittleEndian

type PageID int64

type RowID int64

const RootSchemaPageID PageID = 0

var dblockMagic = []byte("dblock")

func encodeMetadata() []byte {
	metadata := make([]byte, MetadataLength)
	copy(metadata, dblockMagic)
	return metadata
}

func joinHeaderPageData(data []byte) []byte {
	result := make([]byte, MetadataLength+len(data))
	copy(result, encodeMetadata())
	copy(result[MetadataLength:], data)
	return result
}

func splitHeaderPageData(data []byte) ([]byte, []byte) {
	if len(data) < MetadataLength {
		return nil, data
	}
	return data[:MetadataLength], data[MetadataLength:]
}

const PageSize = 8 * 1024

const (
	magicLeaf  = 0x4C454146 // "LEAF"
	magicInt   = 0x494E5445 // "INTE"
	headerSize = 32         // magic(4) + isLeaf(1) + numKeys(2) + next(8) + pad(5)

	leafSlotSize     = 8  // keyOff, keyLen, valOff, valLen (uint16 each)
	internalSlotSize = 12 // keyOff, keyLen (uint16 each) + childID (uint64)
)

// ---- Page header ----
// [0:4]   magic
// [4:5]   isLeaf (0/1)
// [5:7]   numKeys (uint16)
// [7:15]  next leaf pageID (uint64) -- 0 = none, leaf only
// [15:32] padding
// followed by: for each entry, a slot table + variable-length data,
// packed from the end of the page backward (classic slotted page layout).

// slot describes one key (and value, if leaf) stored in the page.
type slot struct {
	keyOff, keyLen     uint16
	valueOff, valueLen uint16 // leaf only
	childID            PageID // internal only
}

// size returns the on-page width of the slot's fixed-size table entry.
func (s slot) size(isLeaf bool) int {
	if isLeaf {
		return leafSlotSize
	}
	return internalSlotSize
}

// put writes the slot's fixed-width table entry into buf at slotStart.
func (s slot) put(buf []byte, slotStart int, isLeaf bool) {
	LE.PutUint16(buf[slotStart:], s.keyOff)
	LE.PutUint16(buf[slotStart+2:], s.keyLen)
	if isLeaf {
		LE.PutUint16(buf[slotStart+4:], s.valueOff)
		LE.PutUint16(buf[slotStart+6:], s.valueLen)
	} else {
		LE.PutUint64(buf[slotStart+4:], uint64(s.childID))
	}
}

// getSlot reads the fixed-width slot table entry at slotStart.
func getSlot(buf []byte, slotStart int, isLeaf bool) slot {
	var s slot
	s.keyOff = LE.Uint16(buf[slotStart:])
	s.keyLen = LE.Uint16(buf[slotStart+2:])
	if isLeaf {
		s.valueOff = LE.Uint16(buf[slotStart+4:])
		s.valueLen = LE.Uint16(buf[slotStart+6:])
	} else {
		s.childID = PageID(LE.Uint64(buf[slotStart+4:]))
	}
	return s
}

// putChildID writes just the childID field of an internal slot entry
// (used for the rightmost child, which has no accompanying key).
func putChildID(buf []byte, slotStart int, childID PageID) {
	LE.PutUint64(buf[slotStart+4:], uint64(childID))
}

// getChildID reads just the childID field of an internal slot entry.
func getChildID(buf []byte, slotStart int) PageID {
	return PageID(LE.Uint64(buf[slotStart+4:]))
}

type Page struct {
	ID       PageID
	IsLeaf   bool
	NumKeys  uint16
	NextLeaf PageID
	Keys     [][]byte
	Values   [][]byte // leaf only, parallel to Keys
	Children []PageID // internal only, len(Children) == len(Keys)+1
}

// Layout: header | slot table (fixed-width, front-to-back) | data (back-to-front)
func (p *Page) Encode() ([]byte, error) {
	pageSize := PageSize

	if p.ID == RootSchemaPageID {
		pageSize = PageSize - MetadataLength
	}

	var s slot
	slotSize := s.size(p.IsLeaf)

	numSlots := len(p.Keys)
	if !p.IsLeaf {
		numSlots++ // rightmost child has no key but needs a slot entry
	}
	slotTableEnd := headerSize + numSlots*slotSize

	totalData := 0
	for i, key := range p.Keys {
		totalData += len(key)
		if p.IsLeaf {
			totalData += len(p.Values[i])
		}
	}

	if slotTableEnd+totalData > pageSize {
		return nil, errors.New("page overflow: node too large for page size")
	}

	buf := make([]byte, pageSize)

	magic := uint32(magicInt)
	if p.IsLeaf {
		magic = magicLeaf
	}
	LE.PutUint32(buf[0:4], magic)
	if p.IsLeaf {
		buf[4] = 1
	}
	LE.PutUint16(buf[5:7], p.NumKeys)
	LE.PutUint64(buf[7:15], uint64(p.NextLeaf))

	slotTableStart := headerSize
	dataEnd := pageSize // data grows backward from here

	for i, key := range p.Keys {
		dataEnd -= len(key)
		s := slot{keyOff: uint16(dataEnd), keyLen: uint16(len(key))}
		copy(buf[s.keyOff:int(s.keyOff)+len(key)], key)

		if p.IsLeaf {
			val := p.Values[i]
			dataEnd -= len(val)
			s.valueOff = uint16(dataEnd)
			s.valueLen = uint16(len(val))
			copy(buf[s.valueOff:int(s.valueOff)+len(val)], val)
		} else {
			s.childID = p.Children[i]
		}

		slotStart := slotTableStart + i*slotSize
		s.put(buf, slotStart, p.IsLeaf)
	}

	// internal nodes have one extra child (rightmost)
	if !p.IsLeaf && len(p.Children) > len(p.Keys) {
		lastSlot := slotTableStart + len(p.Keys)*slotSize
		putChildID(buf, lastSlot, p.Children[len(p.Children)-1])
	}

	return buf, nil
}

var ErrEmptyPage = errors.New("page is uninitialized")

func Decode(buf []byte) (*Page, error) {
	magic := LE.Uint32(buf[0:4])

	if magic == 0 {
		return nil, ErrEmptyPage
	}

	isLeaf := buf[4] == 1
	if magic != magicLeaf && magic != magicInt {
		return nil, errors.New("bad page magic")
	}

	numKeys := LE.Uint16(buf[5:7])
	next := PageID(LE.Uint64(buf[7:15]))

	p := &Page{IsLeaf: isLeaf, NumKeys: numKeys, NextLeaf: next}

	var s slot
	slotSize := s.size(isLeaf)
	slotTableStart := headerSize

	for i16 := range numKeys {
		i := int(i16)
		slotStart := slotTableStart + i*slotSize
		if slotStart+slotSize > len(buf) {
			return nil, errors.New("slot extends past page")
		}
		s := getSlot(buf, slotStart, isLeaf)

		keyEnd := int(s.keyOff) + int(s.keyLen)
		if keyEnd > len(buf) {
			return nil, errors.New("key extends past page")
		}
		key := make([]byte, s.keyLen)
		copy(key, buf[s.keyOff:keyEnd])
		p.Keys = append(p.Keys, key)

		if isLeaf {
			valEnd := int(s.valueOff) + int(s.valueLen)
			if valEnd > len(buf) {
				return nil, errors.New("value extends past page")
			}
			val := make([]byte, s.valueLen)
			copy(val, buf[s.valueOff:valEnd])
			p.Values = append(p.Values, val)
		} else {
			p.Children = append(p.Children, s.childID)
		}
	}

	if p.IsLeaf {
		if len(p.Keys) != len(p.Values) {
			return nil, fmt.Errorf("decode: %d keys != %d values", len(p.Keys), len(p.Values))
		}
	}

	if !isLeaf {
		lastSlot := slotTableStart + int(numKeys)*slotSize
		if lastSlot+slotSize > len(buf) {
			return nil, errors.New("rightmost child slot extends past page")
		}
		p.Children = append(p.Children, getChildID(buf, lastSlot))
	}

	return p, nil
}

// KeyCompare is the ordering used across the tree — byte-lexicographic by default.
func KeyCompare(a, b []byte) int {
	return bytes.Compare(a, b)
}
