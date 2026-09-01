package login

import (
	"context"
	"errors"
	"testing"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/world"
)

func newTestService() *Service {
	repository := account.NewDevelopmentRepository(
		account.ID(41),
		"dev@example.com",
		"development-only",
		character.ID(73),
	)
	return NewService(repository, world.EntryGrant{Ticket: "opaque-value", Host: "127.0.0.1", Port: 7777})
}

func TestAuthenticateReturnsDefaultCharacterAndWorldEntry(t *testing.T) {
	result, err := newTestService().Authenticate(context.Background(), " DEV@example.com ", "development-only")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountID != account.ID(41) || result.CharacterID != character.ID(73) {
		t.Fatalf("identity=%+v", result)
	}
	if result.World.Ticket != "opaque-value" || result.World.Host != "127.0.0.1" || result.World.Port != 7777 {
		t.Fatalf("world=%+v", result.World)
	}
}

func TestAuthenticateUsesOneGenericFailure(t *testing.T) {
	for _, credentials := range [][2]string{
		{"missing@example.com", "development-only"},
		{"dev@example.com", "wrong-password"},
		{"", ""},
	} {
		_, err := newTestService().Authenticate(context.Background(), credentials[0], credentials[1])
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("credentials=%q error=%v", credentials[0], err)
		}
		if err.Error() != "invalid credentials" {
			t.Fatalf("non-generic error=%q", err)
		}
	}
}
