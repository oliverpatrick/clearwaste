package login

import (
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
)

const (
	OpcodeClientHello   = opcode.ClientHello
	OpcodeServerHello   = opcode.ServerHello
	OpcodeLoginRequest  = opcode.LoginRequest
	OpcodeLoginAccepted = opcode.LoginAccepted
	OpcodeLoginRejected = opcode.LoginRejected
)

type ClientHello struct {
	ProtocolVersion uint16
}

func (ClientHello) Opcode() protocol.Opcode { return OpcodeClientHello }

type ServerHello struct {
	Accepted        bool
	ProtocolVersion uint16
}

func (ServerHello) Opcode() protocol.Opcode { return OpcodeServerHello }

type LoginRequest struct {
	Ticket string
}

func (LoginRequest) Opcode() protocol.Opcode { return OpcodeLoginRequest }

type LoginAccepted struct{}

func (LoginAccepted) Opcode() protocol.Opcode { return OpcodeLoginAccepted }

type LoginRejected struct{}

func (LoginRejected) Opcode() protocol.Opcode { return OpcodeLoginRejected }
