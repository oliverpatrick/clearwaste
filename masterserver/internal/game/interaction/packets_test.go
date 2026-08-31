package interaction

import (
	"errors"
	"reflect"
	"testing"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/game/entity"
)

func TestInteractRequestRoundTripsChopAndMine(t *testing.T) {
	for _, action := range []Action{Chop, Mine} {
		want := InteractRequest{TargetID: entity.ID(4812), Action: action}
		registry := newRegistry(t)
		frame, err := registry.Encode(want)
		if err != nil {
			t.Fatal(err)
		}
		if len(frame.Payload) != 9 {
			t.Fatalf("payload length=%d", len(frame.Payload))
		}
		got, err := registry.Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("message=%#v want=%#v", got, want)
		}
	}
}

func TestInteractRequestRejectsInvalidActions(t *testing.T) {
	for _, action := range []byte{0, 3, 255} {
		payload := []byte{0, 0, 0, 0, 0, 0, 0, 1, action}
		_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: payload})
		if !errors.Is(err, ErrInvalidAction) {
			t.Fatalf("action=%d error=%v", action, err)
		}
	}
}

func TestInteractRequestRejectsZeroTarget(t *testing.T) {
	payload := []byte{0, 0, 0, 0, 0, 0, 0, 0, byte(Chop)}
	_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: payload})
	if !errors.Is(err, ErrInvalidTargetID) {
		t.Fatalf("error=%v", err)
	}
}

func TestInteractRequestRequiresExactlyNineBytes(t *testing.T) {
	truncatedPayload := []byte{0, 0, 0, 0, 0, 0, 0, 1}
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: truncatedPayload})
	if !errors.Is(truncated, protocol.ErrUnderflow) {
		t.Fatalf("truncated error=%v", truncated)
	}
	trailingPayload := []byte{0, 0, 0, 0, 0, 0, 0, 1, byte(Mine), 0}
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: trailingPayload})
	if !errors.Is(trailing, network.ErrTrailingPayload) {
		t.Fatalf("trailing error=%v", trailing)
	}
}

func newRegistry(t *testing.T) *network.Registry {
	t.Helper()
	registry := network.NewRegistry()
	if err := RegisterCodecs(registry); err != nil {
		t.Fatal(err)
	}
	return registry
}
