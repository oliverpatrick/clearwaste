package protocol

import (
	"encoding/binary"
	"math"
)

type Writer struct {
	buf []byte
}

// NewWriter returns a writer with capacity for a typical payload.
func NewWriter(capacity int) *Writer { return &Writer{buf: make([]byte, 0, capacity)} }

func (w *Writer) Uint8(v uint8) { w.buf = append(w.buf, v) }

func (w *Writer) Uint16(v uint16) { w.buf = binary.BigEndian.AppendUint16(w.buf, v) }

func (w *Writer) Uint32(v uint32) { w.buf = binary.BigEndian.AppendUint32(w.buf, v) }

func (w *Writer) Uint64(v uint64) { w.buf = binary.BigEndian.AppendUint64(w.buf, v) }

func (w *Writer) Int32(v int32) { w.Uint32(uint32(v)) }

func (w *Writer) Bool(v bool) {
	if v {
		w.Uint8(1)
		return
	}
	w.Uint8(0)
}

func (w *Writer) String(v string) error { return w.Bytes([]byte(v)) }

func (w *Writer) Bytes(v []byte) error {
	if len(v) > math.MaxUint16 {
		return ErrValueTooLarge
	}
	w.Uint16(uint16(len(v)))
	w.buf = append(w.buf, v...)
	return nil
}

// Buffer returns the encoded payload. It remains owned by the writer.
func (w *Writer) Buffer() []byte { return w.buf }
