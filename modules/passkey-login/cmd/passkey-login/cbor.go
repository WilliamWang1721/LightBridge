package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Minimal CBOR decoder for WebAuthn needs (attestationObject + COSE_Key).
// Supports definite-length: int/negint, bytes, string, array, map, bool/null.
// Tags/indefinite length are not supported.

type cborDecoder struct {
	b []byte
	i int
}

func decodeCBORFrom(b []byte) (any, int, error) {
	d := &cborDecoder{b: b}
	v, err := d.decode()
	if err != nil {
		return nil, 0, err
	}
	return v, d.i, nil
}

func (d *cborDecoder) readByte() (byte, error) {
	if d.i >= len(d.b) {
		return 0, errors.New("cbor: unexpected eof")
	}
	c := d.b[d.i]
	d.i++
	return c, nil
}

func (d *cborDecoder) readN(n int) ([]byte, error) {
	if n < 0 || d.i+n > len(d.b) {
		return nil, errors.New("cbor: unexpected eof")
	}
	out := d.b[d.i : d.i+n]
	d.i += n
	return out, nil
}

func (d *cborDecoder) readUint(ai byte) (uint64, error) {
	switch {
	case ai < 24:
		return uint64(ai), nil
	case ai == 24:
		b, err := d.readByte()
		return uint64(b), err
	case ai == 25:
		buf, err := d.readN(2)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint16(buf)), nil
	case ai == 26:
		buf, err := d.readN(4)
		if err != nil {
			return 0, err
		}
		return uint64(binary.BigEndian.Uint32(buf)), nil
	case ai == 27:
		buf, err := d.readN(8)
		if err != nil {
			return 0, err
		}
		return binary.BigEndian.Uint64(buf), nil
	case ai == 31:
		return 0, errors.New("cbor: indefinite length not supported")
	default:
		return 0, fmt.Errorf("cbor: invalid additional info %d", ai)
	}
}

func (d *cborDecoder) decode() (any, error) {
	ib, err := d.readByte()
	if err != nil {
		return nil, err
	}
	major := ib >> 5
	ai := ib & 0x1f

	switch major {
	case 0: // unsigned
		u, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		if u > uint64(^uint(0)) {
			return nil, errors.New("cbor: int overflow")
		}
		return int64(u), nil
	case 1: // negative
		u, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		// -1 - n
		if u > (1<<63)-1 {
			return nil, errors.New("cbor: negint overflow")
		}
		return int64(-1) - int64(u), nil
	case 2: // bytes
		n, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		if n > uint64(len(d.b)-d.i) {
			return nil, errors.New("cbor: bytes length out of range")
		}
		buf, err := d.readN(int(n))
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(buf))
		copy(out, buf)
		return out, nil
	case 3: // text
		n, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		if n > uint64(len(d.b)-d.i) {
			return nil, errors.New("cbor: string length out of range")
		}
		buf, err := d.readN(int(n))
		if err != nil {
			return nil, err
		}
		return string(buf), nil
	case 4: // array
		n, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		if n > 1<<20 {
			return nil, errors.New("cbor: array too large")
		}
		arr := make([]any, 0, int(n))
		for i := 0; i < int(n); i++ {
			v, err := d.decode()
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case 5: // map
		n, err := d.readUint(ai)
		if err != nil {
			return nil, err
		}
		if n > 1<<20 {
			return nil, errors.New("cbor: map too large")
		}
		m := make(map[any]any, int(n))
		for i := 0; i < int(n); i++ {
			k, err := d.decode()
			if err != nil {
				return nil, err
			}
			v, err := d.decode()
			if err != nil {
				return nil, err
			}
			m[k] = v
		}
		return m, nil
	case 7: // simple/float
		switch ai {
		case 20:
			return false, nil
		case 21:
			return true, nil
		case 22:
			return nil, nil
		default:
			return nil, fmt.Errorf("cbor: unsupported simple type %d", ai)
		}
	default:
		return nil, fmt.Errorf("cbor: unsupported major type %d", major)
	}
}

