package interaction

import (
	"errors"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/game/entity"
)

var (
	// ErrInvalidTargetID means an interaction packet used reserved entity ID zero.
	ErrInvalidTargetID = errors.New("invalid interaction target ID")
	// ErrInvalidAction means an interaction packet contained no supported action.
	ErrInvalidAction = errors.New("invalid interaction action")
)

// Opcode returns the stable InteractRequest wire opcode.
func (InteractRequest) Opcode() protocol.Opcode { return opcode.InteractRequest }

// Gameplay marks InteractRequest for authenticated gameplay delivery.
func (InteractRequest) Gameplay() {}

// RegisterCodecs registers the environmental interaction request codec.
func RegisterCodecs(registry *network.Registry) error {
	return network.Register(registry, opcode.InteractRequest, decodeInteractRequest, encodeInteractRequest)
}

func decodeInteractRequest(reader *protocol.Reader) (InteractRequest, error) {
	target, err := reader.Uint64()
	if err != nil {
		return InteractRequest{}, err
	}
	rawAction, err := reader.Uint8()
	if err != nil {
		return InteractRequest{}, err
	}
	request := InteractRequest{TargetID: entity.ID(target), Action: Action(rawAction)}
	if !request.TargetID.Valid() {
		return InteractRequest{}, ErrInvalidTargetID
	}
	if request.Action <= unspecified || request.Action > Mine {
		return InteractRequest{}, ErrInvalidAction
	}
	return request, nil
}

func encodeInteractRequest(writer *protocol.Writer, request InteractRequest) error {
	if !request.TargetID.Valid() {
		return ErrInvalidTargetID
	}
	if request.Action <= unspecified || request.Action > Mine {
		return ErrInvalidAction
	}
	writer.Uint64(uint64(request.TargetID))
	writer.Uint8(uint8(request.Action))
	return nil
}
