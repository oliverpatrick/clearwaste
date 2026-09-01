package account

import (
	"context"
	"strings"

	"master/clearwaste/internal/character"
)

type DevelopmentRepository struct {
	record Record
}

func NewDevelopmentRepository(id ID, email, password string, characterID character.ID) *DevelopmentRepository {
	return &DevelopmentRepository{record: Record{
		ID:                 id,
		Email:              strings.ToLower(strings.TrimSpace(email)),
		DefaultCharacterID: characterID,
		password:           append([]byte(nil), password...),
	}}
}

func (r *DevelopmentRepository) FindByEmail(ctx context.Context, email string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if email != r.record.Email {
		return Record{}, ErrNotFound
	}
	return r.record, nil
}
