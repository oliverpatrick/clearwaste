package protocol

import "encoding/binary"

type Reader struct {
	buf []byte
	pos int
}

// NewReader returns a reader over one packet payload.
func NewReader(buf []byte) *Reader { return &Reader{buf: buf} }

func (r *Reader) Uint8() (uint8, error) {
	b, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *Reader) Uint16() (uint16, error) {
	b, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func (r *Reader) Uint32() (uint32, error) {
	b, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func (r *Reader) Uint64() (uint64, error) {
	b, err := r.take(8)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

func (r *Reader) Int32() (int32, error) {
	v, err := r.Uint32()
	return int32(v), err
}

func (r *Reader) Bool() (bool, error) {
	v, err := r.Uint8()
	if err != nil {
		return false, err
	}
	switch v {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, ErrInvalidBool
	}
}

func (r *Reader) String() (string, error) {
	b, err := r.Bytes()
	return string(b), err
}

func (r *Reader) Bytes() ([]byte, error) {
	n, err := r.Uint16()
	if err != nil {
		return nil, err
	}
	b, err := r.take(int(n))
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), b...), nil
}

// Remaining reports unread payload bytes.
func (r *Reader) Remaining() int { return len(r.buf) - r.pos }

func (r *Reader) take(n int) ([]byte, error) {
	if n < 0 || n > r.Remaining() {
		return nil, ErrUnderflow
	}
	b := r.buf[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}
