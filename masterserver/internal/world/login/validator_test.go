package login

import (
	"errors"
	"strings"
	"testing"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/engine/network"
)

func TestDevelopmentValidatorMapsTicketToIdentity(t *testing.T) {
	want := network.Identity{AccountID: account.ID(41), CharacterID: character.ID(73)}
	got, err := NewDevelopmentValidator("opaque", want).Validate("opaque")
	if err != nil || got != want {
		t.Fatalf("identity=%+v err=%v", got, err)
	}
}

func TestDevelopmentValidatorRejectsInvalidAndEmptyTickets(t *testing.T) {
	identity := network.Identity{AccountID: account.ID(41), CharacterID: character.ID(73)}
	validator := NewDevelopmentValidator("opaque", identity)
	for _, ticket := range []string{"", "wrong-ticket"} {
		_, err := validator.Validate(ticket)
		if !errors.Is(err, ErrInvalidTicket) {
			t.Fatalf("ticket length=%d error=%v", len(ticket), err)
		}
		if strings.Contains(err.Error(), ticket) && ticket != "" {
			t.Fatal("validation error exposed ticket")
		}
	}
}

func TestDevelopmentValidatorRejectsUnavailableConfiguration(t *testing.T) {
	validIdentity := network.Identity{AccountID: account.ID(41), CharacterID: character.ID(73)}
	tests := []struct {
		name     string
		ticket   string
		identity network.Identity
	}{
		{name: "unconfigured"},
		{name: "missing ticket", identity: validIdentity},
		{name: "missing account", ticket: "opaque", identity: network.Identity{CharacterID: character.ID(73)}},
		{name: "missing character", ticket: "opaque", identity: network.Identity{AccountID: account.ID(41)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewDevelopmentValidator(test.ticket, test.identity).Validate("opaque")
			if !errors.Is(err, ErrValidatorUnavailable) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
