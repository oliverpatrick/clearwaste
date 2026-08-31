package opcode

import (
	"testing"

	"master/clearwaste/internal/engine/network/protocol"
)

func TestValuesAreStableAndUnique(t *testing.T) {
	tests := []struct {
		name string
		got  protocol.Opcode
		want protocol.Opcode
	}{
		{"ClientHello", ClientHello, 1},
		{"ServerHello", ServerHello, 2},
		{"LoginRequest", LoginRequest, 3},
		{"LoginAccepted", LoginAccepted, 4},
		{"LoginRejected", LoginRejected, 5},
		{"MoveRequest", MoveRequest, 6},
		{"SetRunEnabled", SetRunEnabled, 7},
		{"InteractRequest", InteractRequest, 8},
	}

	seen := make(map[protocol.Opcode]string, len(tests))
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s=%d want=%d", test.name, test.got, test.want)
		}
		if previous, exists := seen[test.got]; exists {
			t.Errorf("%s and %s share opcode %d", previous, test.name, test.got)
		}
		seen[test.got] = test.name
	}
}
