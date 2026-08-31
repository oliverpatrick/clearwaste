package protocol

import (
	"errors"
	"math"
	"testing"
)

func TestPrimitiveReaderWriterRoundTrip(t *testing.T) {
	w := NewWriter(64)
	w.Uint8(0x12)
	w.Uint16(0x3456)
	w.Uint32(0x789abcde)
	w.Uint64(0x0123456789abcdef)
	w.Int32(-123456)
	w.Bool(true)
	w.Bool(false)
	if err := w.String("waste"); err != nil {
		t.Fatal(err)
	}
	if err := w.Bytes([]byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}

	r := NewReader(w.Buffer())
	if got, err := r.Uint8(); err != nil || got != 0x12 {
		t.Fatalf("uint8=%x err=%v", got, err)
	}
	if got, err := r.Uint16(); err != nil || got != 0x3456 {
		t.Fatalf("uint16=%x err=%v", got, err)
	}
	if got, err := r.Uint32(); err != nil || got != 0x789abcde {
		t.Fatalf("uint32=%x err=%v", got, err)
	}
	if got, err := r.Uint64(); err != nil || got != 0x0123456789abcdef {
		t.Fatalf("uint64=%x err=%v", got, err)
	}
	if got, err := r.Int32(); err != nil || got != -123456 {
		t.Fatalf("int32=%d err=%v", got, err)
	}
	if got, err := r.Bool(); err != nil || !got {
		t.Fatalf("bool=%t err=%v", got, err)
	}
	if got, err := r.Bool(); err != nil || got {
		t.Fatalf("bool=%t err=%v", got, err)
	}
	if got, err := r.String(); err != nil || got != "waste" {
		t.Fatalf("string=%q err=%v", got, err)
	}
	if got, err := r.Bytes(); err != nil || len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("bytes=%v err=%v", got, err)
	}
	if r.Remaining() != 0 {
		t.Fatalf("remaining=%d", r.Remaining())
	}
}

func TestEmptyLengthPrefixedValuesAreValid(t *testing.T) {
	w := NewWriter(4)
	if err := w.String(""); err != nil {
		t.Fatal(err)
	}
	if err := w.Bytes(nil); err != nil {
		t.Fatal(err)
	}
	r := NewReader(w.Buffer())
	s, err := r.String()
	if err != nil || s != "" {
		t.Fatalf("string=%q err=%v", s, err)
	}
	b, err := r.Bytes()
	if err != nil || len(b) != 0 {
		t.Fatalf("bytes=%v err=%v", b, err)
	}
}

func TestReaderUnderflowReturnsError(t *testing.T) {
	r := NewReader([]byte{1, 2, 3})
	if _, err := r.Uint32(); !errors.Is(err, ErrUnderflow) {
		t.Fatalf("error=%v", err)
	}
}

func TestReaderRejectsInvalidBool(t *testing.T) {
	r := NewReader([]byte{2})
	if _, err := r.Bool(); !errors.Is(err, ErrInvalidBool) {
		t.Fatalf("error=%v", err)
	}
}

func TestWriterRejectsOversizedLengthPrefixedValue(t *testing.T) {
	w := NewWriter(0)
	value := make([]byte, math.MaxUint16+1)
	if err := w.Bytes(value); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("bytes error=%v", err)
	}
	if err := w.String(string(value)); !errors.Is(err, ErrValueTooLarge) {
		t.Fatalf("string error=%v", err)
	}
}

func TestReaderCopiesLengthPrefixedBytes(t *testing.T) {
	encoded := []byte{0, 1, 7}
	got, err := NewReader(encoded).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	encoded[2] = 9
	if got[0] != 7 {
		t.Fatalf("bytes retained input buffer: %v", got)
	}
}
