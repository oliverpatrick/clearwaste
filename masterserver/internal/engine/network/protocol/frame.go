package protocol

// HeaderSize is the fixed frame header length: uint16 opcode and uint32 payload length.
const HeaderSize = 6

// Frame is one decoded wire packet.
type Frame struct {
	Opcode  Opcode
	Payload []byte
}
