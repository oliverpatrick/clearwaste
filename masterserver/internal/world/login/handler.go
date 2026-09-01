package login

import (
	"errors"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/world"
	"master/clearwaste/internal/world/bootstrap"
)

var (
	ErrIllegalMessageState = errors.New("message illegal in session state")
	ErrGameplayBeforeLogin = errors.New("gameplay message before login")
	ErrUnexpectedMessage   = errors.New("unexpected application message")
)

// Handler consumes world-login messages and gates gameplay delivery by state.
type Handler struct {
	protocolVersion uint16
	validator       LoginValidator
	runtime         *world.State
}

// NewHandler returns a state-aware world-login handler.
func NewHandler(protocolVersion uint16, validator LoginValidator, runtime ...*world.State) *Handler {
	h := &Handler{protocolVersion: protocolVersion, validator: validator}
	if len(runtime) > 0 {
		h.runtime = runtime[0]
	}
	return h
}

// Handle implements network.InboundHandler.
func (h *Handler) Handle(session *network.Session, message network.Message) (bool, error) {
	switch message := message.(type) {
	case ClientHello:
		if session.State() != network.StateHandshake {
			return false, ErrIllegalMessageState
		}
		if message.ProtocolVersion != h.protocolVersion {
			return false, session.Send(ServerHello{Accepted: false, ProtocolVersion: h.protocolVersion})
		}
		if err := session.AcceptHandshake(); err != nil {
			return false, err
		}
		return false, session.Send(ServerHello{Accepted: true, ProtocolVersion: h.protocolVersion})

	case LoginRequest:
		if session.State() != network.StateLogin {
			return false, ErrIllegalMessageState
		}
		if h.validator == nil {
			return false, session.Send(LoginRejected{})
		}
		identity, err := h.validator.Validate(message.Ticket)
		if err != nil {
			return false, session.Send(LoginRejected{})
		}
		if err := session.Authenticate(identity); err != nil {
			return false, err
		}
		if err := session.Send(LoginAccepted{}); err != nil {
			return false, err
		}
		if h.runtime != nil {
			character, ok := world.DevelopmentCharacter(identity.CharacterID)
			if !ok {
				return false, session.Send(LoginRejected{})
			}
			local := h.runtime.SpawnPlayer(character)
			session.SetRuntimeEntityID(uint64(local))
			entities := h.runtime.Visible(character.Spawn.X/64, character.Spawn.Z/64, character.Spawn.Plane)
			if err := session.Send(bootstrap.FromEntities(local, character.Spawn.X/64, character.Spawn.Z/64, character.Spawn.Plane, entities)); err != nil {
				return false, err
			}
		}
		return false, nil

	default:
		if _, ok := message.(network.GameplayMessage); !ok {
			return false, ErrUnexpectedMessage
		}
		if session.State() != network.StateGame {
			return false, ErrGameplayBeforeLogin
		}
		return true, nil
	}
}
