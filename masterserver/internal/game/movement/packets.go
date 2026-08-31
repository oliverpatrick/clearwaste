package movement

import (
	"errors"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
)

// ErrInvalidDirection means a movement packet contained no known direction.
var ErrInvalidDirection = errors.New("invalid movement direction")

// Opcode returns the stable MoveRequest wire opcode.
func (MoveRequest) Opcode() protocol.Opcode { return opcode.MoveRequest }

// Gameplay marks MoveRequest for authenticated gameplay delivery.
func (MoveRequest) Gameplay() {}

// Opcode returns the stable SetRunEnabled wire opcode.
func (SetRunEnabled) Opcode() protocol.Opcode { return opcode.SetRunEnabled }

// Gameplay marks SetRunEnabled for authenticated gameplay delivery.
func (SetRunEnabled) Gameplay() {}

// RegisterCodecs registers movement and run request codecs.
func RegisterCodecs(registry *network.Registry) error {
	if err := network.Register(registry, opcode.MoveRequest, decodeMoveRequest, encodeMoveRequest); err != nil {
		return err
	}
	return network.Register(registry, opcode.SetRunEnabled, decodeSetRunEnabled, encodeSetRunEnabled)
}

func decodeMoveRequest(reader *protocol.Reader) (MoveRequest, error) {
	raw, err := reader.Uint8()
	if err != nil {
		return MoveRequest{}, err
	}
	direction := Direction(raw)
	if direction > NorthWest {
		return MoveRequest{}, ErrInvalidDirection
	}
	return MoveRequest{Direction: direction}, nil
}

func encodeMoveRequest(writer *protocol.Writer, request MoveRequest) error {
	if request.Direction > NorthWest {
		return ErrInvalidDirection
	}
	writer.Uint8(uint8(request.Direction))
	return nil
}

func decodeSetRunEnabled(reader *protocol.Reader) (SetRunEnabled, error) {
	enabled, err := reader.Bool()
	return SetRunEnabled{Enabled: enabled}, err
}

func encodeSetRunEnabled(writer *protocol.Writer, request SetRunEnabled) error {
	writer.Bool(request.Enabled)
	return nil
}
