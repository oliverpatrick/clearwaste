package network

import (
	"errors"
	"testing"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
)

func TestSessionStartsInHandshake(t *testing.T) {
	session := newSession(1, 1)
	if session.State() != StateHandshake {
		t.Fatalf("state=%v", session.State())
	}
	if _, ok := session.Identity(); ok {
		t.Fatal("new session has identity")
	}
}

func TestSessionAcceptHandshakeEntersLogin(t *testing.T) {
	session := newSession(1, 1)
	if err := session.AcceptHandshake(); err != nil {
		t.Fatal(err)
	}
	if session.State() != StateLogin {
		t.Fatalf("state=%v", session.State())
	}
	if err := session.AcceptHandshake(); !errors.Is(err, ErrIllegalSessionState) {
		t.Fatalf("repeated handshake error=%v", err)
	}
}

func TestSessionAuthenticationRequiresLogin(t *testing.T) {
	session := newSession(1, 1)
	identity := Identity{AccountID: 41, CharacterID: 73}
	if err := session.Authenticate(identity); !errors.Is(err, ErrIllegalSessionState) {
		t.Fatalf("error=%v", err)
	}
	if _, ok := session.Identity(); ok {
		t.Fatal("identity stored before login state")
	}
}

func TestSessionRejectsInvalidIdentityWithoutPartialState(t *testing.T) {
	for name, identity := range map[string]Identity{
		"missing account":   {CharacterID: character.ID(73)},
		"missing character": {AccountID: account.ID(41)},
	} {
		t.Run(name, func(t *testing.T) {
			session := newSession(1, 1)
			if err := session.AcceptHandshake(); err != nil {
				t.Fatal(err)
			}
			if err := session.Authenticate(identity); !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("error=%v", err)
			}
			if session.State() != StateLogin {
				t.Fatalf("state=%v", session.State())
			}
			if _, ok := session.Identity(); ok {
				t.Fatal("partial identity stored")
			}
		})
	}
}

func TestSessionAuthenticationStoresIdentityAndEntersGame(t *testing.T) {
	session := newSession(1, 1)
	if err := session.AcceptHandshake(); err != nil {
		t.Fatal(err)
	}
	want := Identity{AccountID: account.ID(41), CharacterID: character.ID(73)}
	if err := session.Authenticate(want); err != nil {
		t.Fatal(err)
	}
	got, ok := session.Identity()
	if !ok || got != want || session.State() != StateGame {
		t.Fatalf("state=%v identity=%+v ok=%t", session.State(), got, ok)
	}
}

func TestClosedSessionRejectsTransitionsAndEnqueue(t *testing.T) {
	session := newSession(1, 1)
	session.close()
	if session.State() != StateClosed {
		t.Fatalf("state=%v", session.State())
	}
	if err := session.AcceptHandshake(); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("transition error=%v", err)
	}
	if err := session.enqueue(testMessage{}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("enqueue error=%v", err)
	}
}
