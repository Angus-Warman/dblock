package metadata

import (
	"fmt"
	"hash/crc32"

	"github.com/Angus-Warman/dblock/internal/codec"
)

type Metadata struct {
	Dblock        string // 0-7
	FileVersion   uint32 // 8-11, increments every time file write occurs
	SchemaVersion uint32 // 12-15, increments when dblock_schema changed
	PageSizePower uint8  // 16, PageSize = 2 ^ PageSizePower, default 13 = 8192
	NumberOfPages uint32 // 17-20
	Token         int64  // 21-28
	// Padding 29-95, 67 bytes
	Checksum uint32 // 96-99
}

const NumProperties = 7 // Make sure to increment this

const headerMagic = "dblock01"
const Length = 100
const padding = 67

// PageSize returns the database page size in bytes, 2 ^ PageSizePower.
func (m *Metadata) PageSize() int {
	return 1 << m.PageSizePower
}

func New() *Metadata {
	m := &Metadata{
		Dblock:        headerMagic,
		FileVersion:   1,
		SchemaVersion: 1,
		PageSizePower: 13, // 8 KB
		NumberOfPages: 0,
		Token:         0,
	}

	m.CalculateChecksum()

	return m
}

func (m *Metadata) CalculateChecksum() {
	e := codec.NewEncoder()

	e.PutBytes([]byte(m.Dblock))
	e.PutUint32(m.FileVersion)
	e.PutUint32(m.SchemaVersion)
	e.PutUint8(m.PageSizePower)
	e.PutUint32(m.NumberOfPages)
	e.PutInt64(m.Token)
	e.Pad(padding)
	checksum := crc32.ChecksumIEEE(e.Bytes())

	m.Checksum = checksum
}

func (m *Metadata) Encode() []byte {
	e := codec.NewEncoder()

	e.PutBytes([]byte(m.Dblock))
	e.PutUint32(m.FileVersion)
	e.PutUint32(m.SchemaVersion)
	e.PutUint8(m.PageSizePower)
	e.PutUint32(m.NumberOfPages)
	e.PutInt64(m.Token)
	e.Pad(padding)
	checksum := crc32.ChecksumIEEE(e.Bytes())
	e.PutUint32(checksum)

	return e.Bytes()
}

func Decode(buf []byte) (*Metadata, error) {
	dec := codec.NewDecoder(buf)

	magic, err := dec.GetBytes(len(headerMagic))

	if err != nil {
		return nil, err
	}

	if string(magic) != headerMagic {
		return nil, fmt.Errorf("DB file does not start with %q", headerMagic)
	}

	fileVersion, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	schemaVersion, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	pageSizePower, err := dec.GetUint8()

	if err != nil {
		return nil, err
	}

	numberOfPages, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	token, err := dec.GetInt64()

	if err != nil {
		return nil, err
	}

	// Padding
	_, err = dec.GetBytes(padding)

	if err != nil {
		return nil, err
	}

	bufChecksum, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	m := &Metadata{
		Dblock:        headerMagic,
		FileVersion:   fileVersion,
		SchemaVersion: schemaVersion,
		PageSizePower: pageSizePower,
		NumberOfPages: numberOfPages,
		Token:         token,
	}

	m.CalculateChecksum()

	if bufChecksum != m.Checksum {
		return nil, fmt.Errorf("checksum invalid")
	}

	return m, nil
}
