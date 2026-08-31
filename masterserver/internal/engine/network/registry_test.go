package network

import (
	"errors"
	"testing"

	"master/clearwaste/internal/engine/network/protocol"
)

const testOpcode protocol.Opcode = 1

type testMessage struct{ Value uint32 }

func (testMessage) Opcode() protocol.Opcode { return testOpcode }

type wrongTestMessage struct{}

func (wrongTestMessage) Opcode() protocol.Opcode { return testOpcode }

func TestRegistryRoundTrip(t *testing.T) {
	r := newTestRegistry(t)
	frame, err := r.Encode(testMessage{Value: 42})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	message, ok := got.(testMessage)
	if !ok || message.Value != 42 {
		t.Fatalf("message=%T %+v", got, got)
	}
}

func TestRegistryRejectsDuplicateOpcode(t *testing.T) {
	r := newTestRegistry(t)
	if err := Register(r, testOpcode, decodeTestMessage, encodeTestMessage); !errors.Is(err, ErrDuplicateOpcode) {
		t.Fatalf("error=%v", err)
	}
}

func TestRegistryRejectsUnknownOpcode(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Decode(protocol.Frame{Opcode: 99}); !errors.Is(err, ErrUnknownOpcode) {
		t.Fatalf("error=%v", err)
	}
}

func TestRegistryRejectsWrongOutboundMessageType(t *testing.T) {
	r := newTestRegistry(t)
	if _, err := r.Encode(wrongTestMessage{}); !errors.Is(err, ErrWrongMessageType) {
		t.Fatalf("error=%v", err)
	}
}

func TestRegistryRejectsTrailingPayload(t *testing.T) {
	r := newTestRegistry(t)
	if _, err := r.Decode(protocol.Frame{Opcode: testOpcode, Payload: []byte{0, 0, 0, 42, 9}}); !errors.Is(err, ErrTrailingPayload) {
		t.Fatalf("error=%v", err)
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	if err := Register(r, testOpcode, decodeTestMessage, encodeTestMessage); err != nil {
		t.Fatal(err)
	}
	return r
}

func decodeTestMessage(r *protocol.Reader) (testMessage, error) {
	v, err := r.Uint32()
	return testMessage{Value: v}, err
}

func encodeTestMessage(w *protocol.Writer, message testMessage) error {
	w.Uint32(message.Value)
	return nil
}
