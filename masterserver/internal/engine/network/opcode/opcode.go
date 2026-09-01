// Package opcode defines the stable application-level wire opcode contract.
package opcode

import "master/clearwaste/internal/engine/network/protocol"

const (
	ClientHello   protocol.Opcode = 1
	ServerHello   protocol.Opcode = 2
	LoginRequest  protocol.Opcode = 3
	LoginAccepted protocol.Opcode = 4
	LoginRejected protocol.Opcode = 5

	MoveRequest     protocol.Opcode = 6
	SetRunEnabled   protocol.Opcode = 7
	InteractRequest protocol.Opcode = 8
	WorldBootstrap  protocol.Opcode = 9
)
