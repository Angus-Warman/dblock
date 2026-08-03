package codec

import (
	"encoding/binary"
	"fmt"
	"math"
)

var LE = binary.LittleEndian

type Encodable interface {
	Encode() []byte
}

type Encoder struct {
	backing []byte
}

func NewEncoder() *Encoder {
	return &Encoder{
		backing: []byte{},
	}
}

func (e *Encoder) PutBool(val bool) {
	if val {
		e.PutUint8(1)
	} else {
		e.PutUint8(0)
	}
}

func (e *Encoder) PutUint8(val uint8) {
	e.backing = append(e.backing, val)
}

func (e *Encoder) PutUint16(val uint16) {
	e.backing = LE.AppendUint16(e.backing, val)
}

const maxInt8 = 1<<7 - 1
const maxInt16 = 1<<15 - 1
const maxInt32 = 1<<31 - 1

func (e *Encoder) PutAsUint8(val int) {
	if val > maxInt8 {
		panic(fmt.Errorf("value %v cannot be stored as uint8", val))
	}

	e.PutUint8(uint8(val))
}

func (e *Encoder) PutAsUint16(val int) {
	if val > maxInt16 {
		panic(fmt.Errorf("value %v cannot be stored as uint16", val))
	}

	e.PutUint16(uint16(val))
}

func (e *Encoder) PutUint32(val uint32) {
	e.backing = LE.AppendUint32(e.backing, val)
}

func (e *Encoder) PutAsUint32(val int) {
	if val > maxInt32 {
		panic(fmt.Errorf("value %v cannot be stored as uint32", val))
	}

	e.PutUint32(uint32(val))
}

func (e *Encoder) PutUint64(val uint64) {
	e.backing = LE.AppendUint64(e.backing, val)
}

func (e *Encoder) PutInt16(val int16) {
	e.PutUint16(uint16(val))
}

func (e *Encoder) PutInt32(val int32) {
	e.PutUint32(uint32(val))
}

func (e *Encoder) PutInt64(val int64) {
	e.PutUint64(uint64(val))
}

func (e *Encoder) PutFloat64(val float64) {
	e.PutUint64(math.Float64bits(val))
}

func (e *Encoder) PutBytesWithLength(b []byte) {
	e.PutAsUint32(len(b))
	e.PutBytes(b)
}

func (e *Encoder) PutBytes(b []byte) {
	e.backing = append(e.backing, b...)
}

func (e *Encoder) PutStringWithLength(s string) {
	e.PutAsUint32(len(s))
	e.PutBytes([]byte(s))
}

func (e *Encoder) PutShortStringWithLength(s string) {
	if len(s) > maxInt8 {
		panic(fmt.Errorf("value %v cannot is longer than %v", s, maxInt8))
	}

	e.PutAsUint8(len(s))
	e.PutBytes([]byte(s))
}

func (e *Encoder) Len() int {
	return len(e.backing)
}

func (e *Encoder) Pad(length int) {
	e.backing = append(e.backing, make([]byte, length)...)
}

func (e *Encoder) Bytes() []byte {
	return e.backing
}
