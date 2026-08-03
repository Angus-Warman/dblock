package internal

import (
	"fmt"

	"dblock2/internal/codec"
)

type Metadata struct {
	Dblock              string // 0-5
	DatabaseVersion     uint16 // 6-7, 1
	PageSizePower       uint8  // 8, PageSize = 2 ^ PageSizePower, default 13 = 8192
	NumberOfPages       uint32 // 9-12
	FileChangeCounter   uint32 // 13-16, increments every time the file is changed
	SchemaChangeCounter uint32 // 17-20, increments every time dblock_schema is changed
	TokenValue          uint32 // 21-24
}

const headerMagic = "dblock"

func NewMetadata() *Metadata {
	return &Metadata{
		Dblock:              headerMagic,
		DatabaseVersion:     1,
		PageSizePower:       13,
		NumberOfPages:       0,
		FileChangeCounter:   0,
		SchemaChangeCounter: 0,
		TokenValue:          0,
	}
}

const MetadataLength = 100

func (m *Metadata) Encode() []byte {
	e := codec.NewEncoder()

	e.PutBytes([]byte(m.Dblock))
	e.PutUint16(m.DatabaseVersion)
	e.PutUint8(m.PageSizePower)
	e.PutUint32(m.NumberOfPages)
	e.PutUint32(m.FileChangeCounter)
	e.PutUint32(m.SchemaChangeCounter)
	e.PutUint32(m.TokenValue)
	e.Pad(MetadataLength)

	return e.Bytes()
}

func DecodeMetadata(buf []byte) (*Metadata, error) {
	dec := codec.NewDecoder(buf)

	magic, err := dec.GetBytes(6)

	if err != nil {
		return nil, err
	}

	if string(magic) != headerMagic {
		return nil, fmt.Errorf("DB file does not start with %q", headerMagic)
	}

	databaseVersion, err := dec.GetUint16()

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

	fileChangeCounter, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	schemaChangeCounter, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	tokenValue, err := dec.GetUint32()

	if err != nil {
		return nil, err
	}

	return &Metadata{
		Dblock:              headerMagic,
		DatabaseVersion:     databaseVersion,
		PageSizePower:       pageSizePower,
		NumberOfPages:       numberOfPages,
		FileChangeCounter:   fileChangeCounter,
		SchemaChangeCounter: schemaChangeCounter,
		TokenValue:          tokenValue,
	}, nil
}
