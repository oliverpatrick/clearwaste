package login

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/game/interaction"
	"master/clearwaste/internal/game/movement"
)

func TestHandlerAcceptsValidHandshake(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	if connection.Session().State() != network.StateHandshake {
		t.Fatalf("initial state=%v", connection.Session().State())
	}
	sendLoginMessage(t, peer, registry, ClientHello{ProtocolVersion: 1})
	response := readLoginMessage(t, peer, registry)
	hello, ok := response.(ServerHello)
	if !ok || !hello.Accepted || hello.ProtocolVersion != 1 {
		t.Fatalf("response=%T %+v", response, response)
	}
	if connection.Session().State() != network.StateLogin {
		t.Fatalf("state=%v", connection.Session().State())
	}
	assertInboundEmpty(t, connection.Session())
}

func TestHandlerRejectsUnsupportedProtocolWithoutAdvancing(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	sendLoginMessage(t, peer, registry, ClientHello{ProtocolVersion: 2})
	response := readLoginMessage(t, peer, registry)
	hello, ok := response.(ServerHello)
	if !ok || hello.Accepted || hello.ProtocolVersion != 1 {
		t.Fatalf("response=%T %+v", response, response)
	}
	if connection.Session().State() != network.StateHandshake {
		t.Fatalf("state=%v", connection.Session().State())
	}
	assertInboundEmpty(t, connection.Session())
}

func TestHandlerRejectsRepeatedHello(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeHandshake(t, connection, peer, registry)
	sendLoginMessage(t, peer, registry, ClientHello{ProtocolVersion: 1})
	if err := connection.Wait(); !errors.Is(err, ErrIllegalMessageState) {
		t.Fatalf("error=%v", err)
	}
}

func TestHandlerRejectsLoginBeforeHandshake(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	sendLoginMessage(t, peer, registry, LoginRequest{Ticket: "opaque"})
	if err := connection.Wait(); !errors.Is(err, ErrIllegalMessageState) {
		t.Fatalf("error=%v", err)
	}
}

func TestHandlerValidTicketStoresIdentityAndEntersGame(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeHandshake(t, connection, peer, registry)
	sendLoginMessage(t, peer, registry, LoginRequest{Ticket: "opaque"})
	if response := readLoginMessage(t, peer, registry); response != (LoginAccepted{}) {
		t.Fatalf("response=%T %+v", response, response)
	}
	want := network.Identity{AccountID: account.ID(41), CharacterID: character.ID(73)}
	identity, ok := connection.Session().Identity()
	if !ok || identity != want || connection.Session().State() != network.StateGame {
		t.Fatalf("state=%v identity=%+v ok=%t", connection.Session().State(), identity, ok)
	}
	assertInboundEmpty(t, connection.Session())
}

func TestHandlerFailedLoginStoresNoIdentity(t *testing.T) {
	tests := []struct {
		name      string
		validator LoginValidator
		ticket    string
	}{
		{name: "invalid", validator: configuredValidator(), ticket: "wrong"},
		{name: "empty", validator: configuredValidator(), ticket: ""},
		{name: "unconfigured", validator: NewDevelopmentValidator("", network.Identity{}), ticket: "opaque"},
		{name: "partial", validator: NewDevelopmentValidator("opaque", network.Identity{AccountID: account.ID(41)}), ticket: "opaque"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connection, peer, registry := startLoginConnection(t, test.validator)
			completeHandshake(t, connection, peer, registry)
			sendLoginMessage(t, peer, registry, LoginRequest{Ticket: test.ticket})
			if response := readLoginMessage(t, peer, registry); response != (LoginRejected{}) {
				t.Fatalf("response=%T %+v", response, response)
			}
			if _, ok := connection.Session().Identity(); ok {
				t.Fatal("failed login stored identity")
			}
			if connection.Session().State() != network.StateLogin {
				t.Fatalf("state=%v", connection.Session().State())
			}
			assertInboundEmpty(t, connection.Session())
		})
	}
}

func TestHandlerRejectsGameplayBeforeGame(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	sendLoginMessage(t, peer, registry, gameplayTestMessage{Value: 9})
	if err := connection.Wait(); !errors.Is(err, ErrGameplayBeforeLogin) {
		t.Fatalf("error=%v", err)
	}
}

func TestHandlerDeliversGameplayAfterLogin(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeLogin(t, connection, peer, registry)
	sendLoginMessage(t, peer, registry, gameplayTestMessage{Value: 9})
	select {
	case message := <-connection.Session().Inbound():
		if message != (gameplayTestMessage{Value: 9}) {
			t.Fatalf("message=%T %+v", message, message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("gameplay message not delivered")
	}
}

func TestHandlerRejectsLoginPacketsAfterGame(t *testing.T) {
	for name, message := range map[string]network.Message{
		"hello": ClientHello{ProtocolVersion: 1},
		"login": LoginRequest{Ticket: "opaque"},
	} {
		t.Run(name, func(t *testing.T) {
			connection, peer, registry := startLoginConnection(t, configuredValidator())
			completeLogin(t, connection, peer, registry)
			sendLoginMessage(t, peer, registry, message)
			if err := connection.Wait(); !errors.Is(err, ErrIllegalMessageState) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestHandlerRejectsUnclassifiedMessage(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeLogin(t, connection, peer, registry)
	sendLoginMessage(t, peer, registry, applicationTestMessage{})
	if err := connection.Wait(); !errors.Is(err, ErrUnexpectedMessage) {
		t.Fatalf("error=%v", err)
	}
}

func TestHandlerDisconnectClosesSession(t *testing.T) {
	connection, peer, _ := startLoginConnection(t, configuredValidator())
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := connection.Wait(); !errors.Is(err, io.EOF) {
		t.Fatalf("error=%v", err)
	}
	if connection.Session().State() != network.StateClosed {
		t.Fatalf("state=%v", connection.Session().State())
	}
	if err := connection.Session().AcceptHandshake(); !errors.Is(err, network.ErrSessionClosed) {
		t.Fatalf("transition error=%v", err)
	}
}

func configuredValidator() LoginValidator {
	return NewDevelopmentValidator("opaque", network.Identity{AccountID: account.ID(41), CharacterID: character.ID(73)})
}

func startLoginConnection(t *testing.T, validator LoginValidator) (*network.Connection, net.Conn, *network.Registry) {
	return startLoginConnectionWithCapacity(t, validator, 4)
}

func startLoginConnectionWithCapacity(t *testing.T, validator LoginValidator, inboundCapacity int) (*network.Connection, net.Conn, *network.Registry) {
	t.Helper()
	registry := newLoginRegistry(t)
	if err := movement.RegisterCodecs(registry); err != nil {
		t.Fatal(err)
	}
	if err := interaction.RegisterCodecs(registry); err != nil {
		t.Fatal(err)
	}
	if err := network.Register(registry, 100, decodeGameplayTestMessage, encodeGameplayTestMessage); err != nil {
		t.Fatal(err)
	}
	if err := network.Register(registry, 101, decodeApplicationTestMessage, encodeApplicationTestMessage); err != nil {
		t.Fatal(err)
	}
	server, peer := net.Pipe()
	connection := network.NewConnection(1, server, registry, network.ConnectionConfig{
		MaxPayloadSize:        1024,
		InboundQueueCapacity:  inboundCapacity,
		OutboundQueueCapacity: 4,
	}, NewHandler(1, validator))
	connection.Start()
	t.Cleanup(func() {
		_ = peer.Close()
		_ = connection.Close()
		_ = connection.Wait()
	})
	return connection, peer, registry
}

func completeHandshake(t *testing.T, connection *network.Connection, peer net.Conn, registry *network.Registry) {
	t.Helper()
	sendLoginMessage(t, peer, registry, ClientHello{ProtocolVersion: 1})
	if response := readLoginMessage(t, peer, registry); response != (ServerHello{Accepted: true, ProtocolVersion: 1}) {
		t.Fatalf("response=%T %+v", response, response)
	}
	if connection.Session().State() != network.StateLogin {
		t.Fatalf("state=%v", connection.Session().State())
	}
}

func completeLogin(t *testing.T, connection *network.Connection, peer net.Conn, registry *network.Registry) {
	t.Helper()
	completeHandshake(t, connection, peer, registry)
	sendLoginMessage(t, peer, registry, LoginRequest{Ticket: "opaque"})
	if response := readLoginMessage(t, peer, registry); response != (LoginAccepted{}) {
		t.Fatalf("response=%T %+v", response, response)
	}
}

func sendLoginMessage(t *testing.T, peer net.Conn, registry *network.Registry, message network.Message) {
	t.Helper()
	frame, err := registry.Encode(message)
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.EncodeFrame(peer, frame, 1024); err != nil {
		t.Fatal(err)
	}
}

func readLoginMessage(t *testing.T, peer net.Conn, registry *network.Registry) network.Message {
	t.Helper()
	frame, err := protocol.DecodeFrame(peer, 1024)
	if err != nil {
		t.Fatal(err)
	}
	message, err := registry.Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func assertInboundEmpty(t *testing.T, session *network.Session) {
	t.Helper()
	select {
	case message := <-session.Inbound():
		t.Fatalf("message reached gameplay inbound: %T %+v", message, message)
	default:
	}
}

type gameplayTestMessage struct{ Value uint8 }

func (gameplayTestMessage) Opcode() protocol.Opcode { return 100 }
func (gameplayTestMessage) Gameplay()               {}

func decodeGameplayTestMessage(reader *protocol.Reader) (gameplayTestMessage, error) {
	value, err := reader.Uint8()
	return gameplayTestMessage{Value: value}, err
}

func encodeGameplayTestMessage(writer *protocol.Writer, message gameplayTestMessage) error {
	writer.Uint8(message.Value)
	return nil
}

type applicationTestMessage struct{}

func (applicationTestMessage) Opcode() protocol.Opcode { return 101 }

func decodeApplicationTestMessage(*protocol.Reader) (applicationTestMessage, error) {
	return applicationTestMessage{}, nil
}

func encodeApplicationTestMessage(*protocol.Writer, applicationTestMessage) error { return nil }
