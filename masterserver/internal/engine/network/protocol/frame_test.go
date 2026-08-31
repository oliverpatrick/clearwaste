package protocol

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestEncodeDecodeFrameRoundTrip(t *testing.T) {
	frame := Frame{Opcode: 0x1234, Payload: []byte{5, 6, 7}}
	var wire bytes.Buffer
	if err := EncodeFrame(&wire, frame, 1024); err != nil {
		t.Fatal(err)
	}
	wantWire := []byte{0x12, 0x34, 0, 0, 0, 3, 5, 6, 7}
	if !bytes.Equal(wire.Bytes(), wantWire) {
		t.Fatalf("wire=%v want=%v", wire.Bytes(), wantWire)
	}
	got, err := DecodeFrame(&wire, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, frame) {
		t.Fatalf("frame=%+v want=%+v", got, frame)
	}
}

func TestDecodeFrameHandlesFragmentedInput(t *testing.T) {
	wire := &oneByteReader{r: bytes.NewReader([]byte{0, 2, 0, 0, 0, 2, 8, 9})}
	got, err := DecodeFrame(wire, 16)
	if err != nil {
		t.Fatal(err)
	}
	if got.Opcode != 2 || !bytes.Equal(got.Payload, []byte{8, 9}) {
		t.Fatalf("frame=%+v", got)
	}
}

func TestDecodeFrameReadsMultipleFramesFromOneStream(t *testing.T) {
	wire := bytes.NewReader([]byte{
		0, 1, 0, 0, 0, 1, 7,
		0, 2, 0, 0, 0, 2, 8, 9,
	})
	first, err := DecodeFrame(wire, 16)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecodeFrame(wire, 16)
	if err != nil {
		t.Fatal(err)
	}
	if first.Opcode != 1 || !bytes.Equal(first.Payload, []byte{7}) || second.Opcode != 2 || !bytes.Equal(second.Payload, []byte{8, 9}) {
		t.Fatalf("frames=%+v %+v", first, second)
	}
}

func TestDecodeFrameDistinguishesEOFAndTruncation(t *testing.T) {
	if _, err := DecodeFrame(bytes.NewReader(nil), 1024); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream error=%v", err)
	}
	if _, err := DecodeFrame(bytes.NewReader([]byte{0}), 1024); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial header error=%v", err)
	}
	if _, err := DecodeFrame(bytes.NewReader([]byte{0, 1, 0, 0, 0, 2, 9}), 1024); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial payload error=%v", err)
	}
}

func TestDecodeFrameRejectsOversizedPayloadBeforeReadingIt(t *testing.T) {
	_, err := DecodeFrame(bytes.NewReader([]byte{0, 1, 0, 0, 4, 1}), 1024)
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("error=%v", err)
	}
}

func TestFrameAcceptsConfiguredMaximumPayload(t *testing.T) {
	frame := Frame{Opcode: 3, Payload: []byte{1, 2, 3, 4}}
	var wire bytes.Buffer
	if err := EncodeFrame(&wire, frame, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeFrame(&wire, 4); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeFrameRejectsOversizedPayload(t *testing.T) {
	if err := EncodeFrame(io.Discard, Frame{Payload: make([]byte, 5)}, 4); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("error=%v", err)
	}
}

type oneByteReader struct{ r io.Reader }

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.r.Read(p)
}
