// Package movement defines movement intent crossing the network/game boundary.
package movement

// Direction is one step on the logical eight-direction grid.
type Direction uint8

const (
	North Direction = iota
	NorthEast
	East
	SouthEast
	South
	SouthWest
	West
	NorthWest
)

// MoveRequest asks the simulation to attempt one step in Direction.
type MoveRequest struct {
	Direction Direction
}

// SetRunEnabled asks the simulation to use the requested run mode.
type SetRunEnabled struct {
	Enabled bool
}
