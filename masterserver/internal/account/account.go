package account

import (
	"context"
	"crypto/subtle"
	"errors"

	"master/clearwaste/internal/character"
)

var ErrNotFound = errors.New("account not found")

type Record struct {
	ID                 ID
	Email              string
	DefaultCharacterID character.ID
	password           []byte
}

func (r Record) PasswordMatches(password string) bool {
	return subtle.ConstantTimeCompare(r.password, []byte(password)) == 1
}

type Repository interface {
	FindByEmail(context.Context, string) (Record, error)
}
