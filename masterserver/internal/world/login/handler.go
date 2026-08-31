package login

import (
	"errors"

	"master/clearwaste/internal/engine/network"
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
}

// NewHandler returns a state-aware world-login handler.
func NewHandler(protocolVersion uint16, validator LoginValidator) *Handler {
	return &Handler{protocolVersion: protocolVersion, validator: validator}
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
		return false, session.Send(LoginAccepted{})

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
