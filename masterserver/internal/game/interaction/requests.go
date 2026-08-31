// Package interaction defines environmental interaction intent.
package interaction

import "master/clearwaste/internal/game/entity"

// Action identifies the requested environmental interaction.
type Action uint8

const (
	unspecified Action = iota
	Chop
	Mine
)

// InteractRequest asks the simulation to attempt Action against TargetID.
type InteractRequest struct {
	TargetID entity.ID
	Action   Action
}
