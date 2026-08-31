package network

import (
	"errors"
	"fmt"
	"sync"

	"master/clearwaste/internal/engine/network/protocol"
)

var (
	ErrUnknownOpcode    = errors.New("unknown opcode")
	ErrDuplicateOpcode  = errors.New("duplicate opcode")
	ErrWrongMessageType = errors.New("wrong message type")
	ErrTrailingPayload  = errors.New("trailing payload bytes")
)

// Message is a typed packet accepted or produced by a registered codec.
type Message interface {
	Opcode() protocol.Opcode
}

type codec struct {
	decode func(*protocol.Reader) (Message, error)
	encode func(*protocol.Writer, Message) error
}

// Registry maps wire opcodes to typed packet codecs.
type Registry struct {
	mu     sync.RWMutex
	codecs map[protocol.Opcode]codec
}

// NewRegistry returns an empty codec registry.
func NewRegistry() *Registry {
	return &Registry{codecs: make(map[protocol.Opcode]codec)}
}

// Register adds one typed packet codec.
func Register[T Message](registry *Registry, opcode protocol.Opcode, decode func(*protocol.Reader) (T, error), encode func(*protocol.Writer, T) error) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.codecs[opcode]; exists {
		return fmt.Errorf("%w: %d", ErrDuplicateOpcode, opcode)
	}
	registry.codecs[opcode] = codec{
		decode: func(reader *protocol.Reader) (Message, error) {
			return decode(reader)
		},
		encode: func(writer *protocol.Writer, message Message) error {
			typed, ok := message.(T)
			if !ok {
				return fmt.Errorf("%w for opcode %d", ErrWrongMessageType, opcode)
			}
			return encode(writer, typed)
		},
	}
	return nil
}

// Decode converts a framed payload to its registered typed message.
func (r *Registry) Decode(frame protocol.Frame) (Message, error) {
	registered, ok := r.lookup(frame.Opcode)
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnknownOpcode, frame.Opcode)
	}
	reader := protocol.NewReader(frame.Payload)
	message, err := registered.decode(reader)
	if err != nil {
		return nil, fmt.Errorf("decode opcode %d: %w", frame.Opcode, err)
	}
	if reader.Remaining() != 0 {
		return nil, fmt.Errorf("%w for opcode %d", ErrTrailingPayload, frame.Opcode)
	}
	if message.Opcode() != frame.Opcode {
		return nil, fmt.Errorf("%w for opcode %d", ErrWrongMessageType, frame.Opcode)
	}
	return message, nil
}

// Encode converts a typed message to a frame.
func (r *Registry) Encode(message Message) (protocol.Frame, error) {
	opcode := message.Opcode()
	registered, ok := r.lookup(opcode)
	if !ok {
		return protocol.Frame{}, fmt.Errorf("%w: %d", ErrUnknownOpcode, opcode)
	}
	writer := protocol.NewWriter(64)
	if err := registered.encode(writer, message); err != nil {
		return protocol.Frame{}, fmt.Errorf("encode opcode %d: %w", opcode, err)
	}
	return protocol.Frame{Opcode: opcode, Payload: writer.Buffer()}, nil
}

func (r *Registry) lookup(opcode protocol.Opcode) (codec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registered, ok := r.codecs[opcode]
	return registered, ok
}
