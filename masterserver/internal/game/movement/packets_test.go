package movement

import (
	"errors"
	"reflect"
	"testing"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
)

func TestMoveRequestRoundTripsAllDirections(t *testing.T) {
	directions := []Direction{North, NorthEast, East, SouthEast, South, SouthWest, West, NorthWest}
	for value, direction := range directions {
		if uint8(direction) != uint8(value) {
			t.Fatalf("direction %d=%d want=%d", value, direction, value)
		}
		assertRoundTrip(t, MoveRequest{Direction: direction})
	}
}

func TestMoveRequestRejectsInvalidDirection(t *testing.T) {
	for _, value := range []byte{8, 255} {
		_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest, Payload: []byte{value}})
		if !errors.Is(err, ErrInvalidDirection) {
			t.Fatalf("direction=%d error=%v", value, err)
		}
	}
}

func TestMoveRequestRequiresExactlyOneByte(t *testing.T) {
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest})
	if !errors.Is(truncated, protocol.ErrUnderflow) {
		t.Fatalf("truncated error=%v", truncated)
	}
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest, Payload: []byte{0, 0}})
	if !errors.Is(trailing, network.ErrTrailingPayload) {
		t.Fatalf("trailing error=%v", trailing)
	}
}

func TestSetRunEnabledDecodesZeroAndOne(t *testing.T) {
	for _, test := range []struct {
		wire byte
		want bool
	}{{0, false}, {1, true}} {
		message, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{test.wire}})
		if err != nil {
			t.Fatal(err)
		}
		if got := message.(SetRunEnabled).Enabled; got != test.want {
			t.Fatalf("enabled=%t want=%t", got, test.want)
		}
	}
}

func TestSetRunEnabledRejectsInvalidOrWrongLengthPayload(t *testing.T) {
	for _, value := range []byte{2, 255} {
		_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{value}})
		if !errors.Is(err, protocol.ErrInvalidBool) {
			t.Fatalf("value=%d error=%v", value, err)
		}
	}
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled})
	if !errors.Is(truncated, protocol.ErrUnderflow) {
		t.Fatalf("truncated error=%v", truncated)
	}
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{1, 0}})
	if !errors.Is(trailing, network.ErrTrailingPayload) {
		t.Fatalf("trailing error=%v", trailing)
	}
}

func assertRoundTrip(t *testing.T, want network.Message) {
	t.Helper()
	registry := newRegistry(t)
	frame, err := registry.Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := registry.Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("message=%#v want=%#v", got, want)
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
