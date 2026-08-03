package codec

import (
	"fmt"
	"math"
)

type Decoder struct {
	pos     int
	backing []byte
}

func NewDecoder(buf []byte) *Decoder {
	return &Decoder{
		pos:     0,
		backing: buf,
	}
}

func (d *Decoder) check(n int) error {
	if d.pos+n > len(d.backing) {
		return fmt.Errorf("need %d bytes at offset %d, but only %d bytes remain", n, d.pos, len(d.backing)-int(d.pos))
	}
	return nil
}

func (d *Decoder) GetUint8() (uint8, error) {
	if err := d.check(1); err != nil {
		return 0, err
	}
	val := d.backing[d.pos]
	d.pos++
	return val, nil
}

func (d *Decoder) GetUint16() (uint16, error) {
	if err := d.check(2); err != nil {
		return 0, err
	}
	val := LE.Uint16(d.backing[d.pos:])
	d.pos += 2
	return val, nil
}

func (d *Decoder) GetAsUint16() (int, error) {
	v, err := d.GetUint16()
	return int(v), err
}

func (d *Decoder) GetUint32() (uint32, error) {
	if err := d.check(4); err != nil {
		return 0, err
	}
	val := LE.Uint32(d.backing[d.pos:])
	d.pos += 4
	return val, nil
}

func (d *Decoder) GetAsUint8() (int, error) {
	v, err := d.GetUint8()
	return int(v), err
}

func (d *Decoder) GetAsUint32() (int, error) {
	v, err := d.GetUint32()
	return int(v), err
}

func (d *Decoder) GetUint64() (uint64, error) {
	if err := d.check(8); err != nil {
		return 0, err
	}
	val := LE.Uint64(d.backing[d.pos:])
	d.pos += 8
	return val, nil
}

func (d *Decoder) GetInt16() (int16, error) {
	v, err := d.GetUint16()
	return int16(v), err
}

func (d *Decoder) GetInt32() (int32, error) {
	v, err := d.GetUint32()
	return int32(v), err
}

func (d *Decoder) GetInt64() (int64, error) {
	v, err := d.GetUint64()
	return int64(v), err
}

func (d *Decoder) GetFloat32() (float32, error) {
	v, err := d.GetUint32()
	return math.Float32frombits(v), err
}

func (d *Decoder) GetFloat64() (float64, error) {
	v, err := d.GetUint64()
	return math.Float64frombits(v), err
}

func (d *Decoder) GetBytes(n int) ([]byte, error) {
	if err := d.check(n); err != nil {
		return nil, err
	}
	b := d.backing[d.pos : d.pos+n]
	d.pos += n
	return b, nil
}

func (d *Decoder) GetStringWithLength() (string, error) {
	n, err := d.GetAsUint32()
	if err != nil {
		return "", err
	}
	b, err := d.GetBytes(int(n))
	return string(b), err
}

func (d *Decoder) GetShortStringWithLength() (string, error) {
	n, err := d.GetAsUint8()
	if err != nil {
		return "", err
	}
	b, err := d.GetBytes(int(n))
	return string(b), err
}

func (d *Decoder) GetBytesWithLength() ([]byte, error) {
	n, err := d.GetAsUint32()
	if err != nil {
		return nil, err
	}
	return d.GetBytes(int(n))
}

func (d *Decoder) Remaining() int {
	return len(d.backing) - d.pos
}

func (d *Decoder) GetBool() (bool, error) {
	val, err := d.GetUint8()

	if err != nil {
		return false, err
	}

	return val == 1, nil
}
