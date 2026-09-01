package account

import (
	"context"
	"strings"

	"master/clearwaste/internal/character"
)

type DevelopmentRepository struct {
	records []Record
}

func NewDevelopmentRepository(id ID, email, password string, characterID character.ID) *DevelopmentRepository {
	return &DevelopmentRepository{records: []Record{{
		ID:                 id,
		Email:              strings.ToLower(strings.TrimSpace(email)),
		DefaultCharacterID: characterID,
		password:           append([]byte(nil), password...),
	}}}
}

func (r *DevelopmentRepository) Add(id ID, email, password string, characterID character.ID) {
	r.records = append(r.records, Record{ID: id, Email: strings.ToLower(strings.TrimSpace(email)), DefaultCharacterID: characterID, password: []byte(password)})
}

func (r *DevelopmentRepository) FindByEmail(ctx context.Context, email string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	for _, record := range r.records {
		if email == record.Email {
			return record, nil
		}
	}
	return Record{}, ErrNotFound
}
