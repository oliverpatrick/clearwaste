package protocol

import (
	"encoding/binary"
	"io"
)

// EncodeFrame writes one complete big-endian frame.
func EncodeFrame(w io.Writer, frame Frame, maxPayloadSize uint32) error {
	if uint64(len(frame.Payload)) > uint64(maxPayloadSize) {
		return ErrPayloadTooLarge
	}
	var header [HeaderSize]byte
	binary.BigEndian.PutUint16(header[:2], uint16(frame.Opcode))
	binary.BigEndian.PutUint32(header[2:], uint32(len(frame.Payload)))
	if err := writeFull(w, header[:]); err != nil {
		return err
	}
	return writeFull(w, frame.Payload)
}

func writeFull(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}
