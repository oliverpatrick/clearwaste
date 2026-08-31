package network

import (
	"errors"
	"sync"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
)

var (
	ErrIllegalSessionState = errors.New("illegal session state")
	ErrInvalidIdentity     = errors.New("invalid session identity")
	ErrSessionClosed       = errors.New("session closed")
)

// SessionState is the authoritative protocol phase for a connection.
type SessionState uint8

const (
	StateHandshake SessionState = iota
	StateLogin
	StateGame
	StateClosed
)

// Identity is the authenticated account and selected character for a session.
type Identity struct {
	AccountID   account.ID
	CharacterID character.ID
}

// Valid reports whether both required identity fields are present.
func (i Identity) Valid() bool { return i.AccountID != 0 && i.CharacterID != 0 }

// ConnectionID uniquely identifies one accepted connection for this process lifetime.
type ConnectionID uint64

// Session is the application-facing input boundary for one connection.
type Session struct {
	id      ConnectionID
	inbound chan Message

	mu          sync.Mutex
	state       SessionState
	identity    Identity
	hasIdentity bool
	send        func(Message) error
}

func newSession(id ConnectionID, capacity int) *Session {
	return &Session{id: id, inbound: make(chan Message, capacity), state: StateHandshake}
}

// ID returns the connection identity associated with this session.
func (s *Session) ID() ConnectionID { return s.id }

// Inbound returns decoded messages for the application to drain.
func (s *Session) Inbound() <-chan Message { return s.inbound }

// State returns the session's current protocol state.
func (s *Session) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Identity returns the authenticated identity after successful login.
func (s *Session) Identity() (Identity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.identity, s.hasIdentity
}

// AcceptHandshake advances a successfully negotiated session to Login.
func (s *Session) AcceptHandshake() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateClosed {
		return ErrSessionClosed
	}
	if s.state != StateHandshake {
		return ErrIllegalSessionState
	}
	s.state = StateLogin
	return nil
}

// Authenticate atomically stores a complete identity and enters Game.
func (s *Session) Authenticate(identity Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateClosed {
		return ErrSessionClosed
	}
	if s.state != StateLogin {
		return ErrIllegalSessionState
	}
	if !identity.Valid() {
		return ErrInvalidIdentity
	}
	s.identity = identity
	s.hasIdentity = true
	s.state = StateGame
	return nil
}

// Send uses the connection's existing bounded outbound path.
func (s *Session) Send(message Message) error {
	s.mu.Lock()
	if s.state == StateClosed {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	send := s.send
	s.mu.Unlock()
	if send == nil {
		return ErrConnectionClosed
	}
	return send(message)
}

func (s *Session) setSender(send func(Message) error) {
	s.mu.Lock()
	s.send = send
	s.mu.Unlock()
}

func (s *Session) enqueue(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateClosed {
		return ErrSessionClosed
	}
	select {
	case s.inbound <- message:
		return nil
	default:
		return ErrInboundBackpressure
	}
}

func (s *Session) close() {
	s.mu.Lock()
	s.state = StateClosed
	s.mu.Unlock()
}
