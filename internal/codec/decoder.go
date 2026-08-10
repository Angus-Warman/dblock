package codec

import (
	"fmt"
	"math"
)

type Decoder struct {
	target  string
	pos     int
	backing []byte
}

func NewDecoder(target string, buf []byte) *Decoder {
	return &Decoder{
		target:  target,
		pos:     0,
		backing: buf,
	}
}

func (d *Decoder) check(context string, need int) error {
	total := len(d.backing)
	remaining := total - d.pos
	missing := need - remaining
	if missing > 0 {
		return fmt.Errorf("decode %v: %v: missing %d bytes, %d bytes remain", d.target, context, missing, remaining)
	}
	return nil
}

func (d *Decoder) GetUint8() (uint8, error) {
	if err := d.check("get uint8", 1); err != nil {
		return 0, err
	}
	val := d.backing[d.pos]
	d.pos++
	return val, nil
}

func (d *Decoder) GetUint16() (uint16, error) {
	if err := d.check("get uint16", 2); err != nil {
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
	if err := d.check("get uint16", 4); err != nil {
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
	if err := d.check("get uint64", 8); err != nil {
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
	if err := d.check("get float32", 4); err != nil {
		return 0, err
	}

	v, err := d.GetUint32()
	return math.Float32frombits(v), err
}

func (d *Decoder) GetFloat64() (float64, error) {
	if err := d.check("get float64", 4); err != nil {
		return 0, err
	}

	v, err := d.GetUint64()
	return math.Float64frombits(v), err
}

func (d *Decoder) GetBytes(n int) ([]byte, error) {
	if err := d.check("get bytes", n); err != nil {
		return nil, err
	}

	b := d.backing[d.pos : d.pos+n]
	d.pos += n
	return b, nil
}

func (d *Decoder) GetStringWithLength() (string, error) {
	if err := d.check("get string length", 4); err != nil {
		return "", err
	}

	n, err := d.GetAsUint32()
	if err != nil {
		return "", err
	}

	if err := d.check("get string", n); err != nil {
		return "", err
	}

	b, err := d.GetBytes(int(n))

	return string(b), err
}

func (d *Decoder) GetShortStringWithLength() (string, error) {
	if err := d.check("get string length", 1); err != nil {
		return "", err
	}

	n, err := d.GetAsUint8()
	if err != nil {
		return "", err
	}

	if err := d.check("get string", n); err != nil {
		return "", err
	}

	b, err := d.GetBytes(int(n))
	return string(b), err
}

func (d *Decoder) GetBytesWithLength() ([]byte, error) {
	if err := d.check("get bytes length", 4); err != nil {
		return nil, err
	}

	n, err := d.GetAsUint32()
	if err != nil {
		return nil, err
	}

	if err := d.check("get bytes", n); err != nil {
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
