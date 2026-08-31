package login

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/protocol"
)

func TestLoginCodecsRoundTrip(t *testing.T) {
	registry := newLoginRegistry(t)
	messages := []network.Message{
		ClientHello{ProtocolVersion: 7},
		ServerHello{Accepted: true, ProtocolVersion: 7},
		LoginRequest{Ticket: "opaque"},
		LoginAccepted{},
		LoginRejected{},
	}
	for _, message := range messages {
		t.Run(reflect.TypeOf(message).Name(), func(t *testing.T) {
			frame, err := registry.Encode(message)
			if err != nil {
				t.Fatal(err)
			}
			got, err := registry.Decode(frame)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, message) {
				t.Fatalf("message=%+v want=%+v", got, message)
			}
		})
	}
}

func TestLoginRequestWireContainsOnlyLengthPrefixedTicket(t *testing.T) {
	frame, err := newLoginRegistry(t).Encode(LoginRequest{Ticket: "opaque"})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0, 6, 'o', 'p', 'a', 'q', 'u', 'e'}
	if !bytes.Equal(frame.Payload, want) {
		t.Fatalf("payload=%v want=%v", frame.Payload, want)
	}
}

func TestLoginRequestAllowsEmptyTicketStructurally(t *testing.T) {
	message, err := newLoginRegistry(t).Decode(protocol.Frame{Opcode: OpcodeLoginRequest, Payload: []byte{0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if message.(LoginRequest).Ticket != "" {
		t.Fatalf("ticket=%q", message.(LoginRequest).Ticket)
	}
}

func TestClientHelloRejectsMalformedPayload(t *testing.T) {
	_, err := newLoginRegistry(t).Decode(protocol.Frame{Opcode: OpcodeClientHello, Payload: []byte{1}})
	if !errors.Is(err, protocol.ErrUnderflow) {
		t.Fatalf("error=%v", err)
	}
}

func TestLoginRequestRejectsAppendedIdentityFields(t *testing.T) {
	payload := []byte{0, 1, 'x', 0, 0, 0, 0, 0, 0, 0, 41}
	_, err := newLoginRegistry(t).Decode(protocol.Frame{Opcode: OpcodeLoginRequest, Payload: payload})
	if !errors.Is(err, network.ErrTrailingPayload) {
		t.Fatalf("error=%v", err)
	}
}

func newLoginRegistry(t *testing.T) *network.Registry {
	t.Helper()
	registry := network.NewRegistry()
	if err := RegisterCodecs(registry); err != nil {
		t.Fatal(err)
	}
	return registry
}
