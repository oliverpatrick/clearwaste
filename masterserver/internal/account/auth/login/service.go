package login

import (
	"context"
	"errors"
	"strings"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/world"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Result struct {
	AccountID   account.ID
	CharacterID character.ID
	World       world.EntryGrant
}

type Service struct {
	accounts account.Repository
	entry    world.EntryGrant
}

func NewService(accounts account.Repository, entry world.EntryGrant) *Service {
	return &Service{accounts: accounts, entry: entry}
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (Result, error) {
	record, err := s.accounts.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !record.PasswordMatches(password) {
		return Result{}, ErrInvalidCredentials
	}
	return Result{AccountID: record.ID, CharacterID: record.DefaultCharacterID, World: s.entry}, nil
}
