package login

import (
	"crypto/subtle"
	"errors"

	"master/clearwaste/internal/engine/network"
)

var (
	ErrInvalidTicket        = errors.New("invalid login ticket")
	ErrValidatorUnavailable = errors.New("login validator unavailable")
)

// LoginValidator resolves an opaque world ticket to authenticated identity.
type LoginValidator interface {
	Validate(ticket string) (network.Identity, error)
}

// DevelopmentValidator maps one configured development ticket to one identity.
// A future signed world-ticket validator replaces this implementation.
type DevelopmentValidator struct {
	ticket    []byte
	identity  network.Identity
	available bool
}

// NewDevelopmentValidator constructs the development-only ticket mapping.
func NewDevelopmentValidator(ticket string, identity network.Identity) *DevelopmentValidator {
	return &DevelopmentValidator{
		ticket:    append([]byte(nil), ticket...),
		identity:  identity,
		available: ticket != "" && identity.Valid(),
	}
}

// Validate returns the configured identity only for an exact non-empty match.
func (v *DevelopmentValidator) Validate(ticket string) (network.Identity, error) {
	if !v.available {
		return network.Identity{}, ErrValidatorUnavailable
	}
	if ticket == "" || subtle.ConstantTimeCompare([]byte(ticket), v.ticket) != 1 {
		return network.Identity{}, ErrInvalidTicket
	}
	return v.identity, nil
}
