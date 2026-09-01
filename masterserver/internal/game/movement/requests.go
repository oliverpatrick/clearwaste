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

func Delta(direction Direction) (int32, int32) {
	switch direction {
	case North:
		return 0, -1
	case NorthEast:
		return 1, -1
	case East:
		return 1, 0
	case SouthEast:
		return 1, 1
	case South:
		return 0, 1
	case SouthWest:
		return -1, 1
	case West:
		return -1, 0
	case NorthWest:
		return -1, -1
	}
	return 0, 0
}
