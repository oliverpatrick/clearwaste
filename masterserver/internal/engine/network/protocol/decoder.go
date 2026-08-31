package protocol

import (
	"encoding/binary"
	"io"
)

// DecodeFrame reads one big-endian frame from a byte stream.
func DecodeFrame(r io.Reader, maxPayloadSize uint32) (Frame, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}

	length := binary.BigEndian.Uint32(header[2:])
	if length > maxPayloadSize {
		return Frame{}, ErrPayloadTooLarge
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, err
	}
	return Frame{
		Opcode:  Opcode(binary.BigEndian.Uint16(header[:2])),
		Payload: payload,
	}, nil
}
