package login

import (
	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/protocol"
)

// RegisterCodecs registers the world-login packet set.
func RegisterCodecs(registry *network.Registry) error {
	if err := network.Register(registry, OpcodeClientHello, decodeClientHello, encodeClientHello); err != nil {
		return err
	}
	if err := network.Register(registry, OpcodeServerHello, decodeServerHello, encodeServerHello); err != nil {
		return err
	}
	if err := network.Register(registry, OpcodeLoginRequest, decodeLoginRequest, encodeLoginRequest); err != nil {
		return err
	}
	if err := network.Register(registry, OpcodeLoginAccepted, decodeLoginAccepted, encodeLoginAccepted); err != nil {
		return err
	}
	return network.Register(registry, OpcodeLoginRejected, decodeLoginRejected, encodeLoginRejected)
}

func decodeClientHello(reader *protocol.Reader) (ClientHello, error) {
	version, err := reader.Uint16()
	return ClientHello{ProtocolVersion: version}, err
}

func encodeClientHello(writer *protocol.Writer, message ClientHello) error {
	writer.Uint16(message.ProtocolVersion)
	return nil
}

func decodeServerHello(reader *protocol.Reader) (ServerHello, error) {
	accepted, err := reader.Bool()
	if err != nil {
		return ServerHello{}, err
	}
	version, err := reader.Uint16()
	return ServerHello{Accepted: accepted, ProtocolVersion: version}, err
}

func encodeServerHello(writer *protocol.Writer, message ServerHello) error {
	writer.Bool(message.Accepted)
	writer.Uint16(message.ProtocolVersion)
	return nil
}

func decodeLoginRequest(reader *protocol.Reader) (LoginRequest, error) {
	ticket, err := reader.String()
	return LoginRequest{Ticket: ticket}, err
}

func encodeLoginRequest(writer *protocol.Writer, message LoginRequest) error {
	return writer.String(message.Ticket)
}

func decodeLoginAccepted(*protocol.Reader) (LoginAccepted, error) { return LoginAccepted{}, nil }

func encodeLoginAccepted(*protocol.Writer, LoginAccepted) error { return nil }

func decodeLoginRejected(*protocol.Reader) (LoginRejected, error) { return LoginRejected{}, nil }

func encodeLoginRejected(*protocol.Writer, LoginRejected) error { return nil }
