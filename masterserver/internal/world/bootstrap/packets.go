package bootstrap

import (
	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/opcode"
	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/game/entity"
	"master/clearwaste/internal/world"
)

const Opcode = opcode.WorldBootstrap

type Snapshot struct {
	ID           entity.ID
	Position     world.Position
	Kind         world.Kind
	DefinitionID uint16
	CharacterID  uint64
	AppearanceID uint16
}
type Message struct {
	LocalEntityID    entity.ID
	RegionX, RegionZ int32
	Plane            uint8
	Entities         []Snapshot
}

func (Message) Opcode() protocol.Opcode { return Opcode }

func RegisterCodecs(registry *network.Registry) error {
	return network.Register(registry, Opcode, decode, encode)
}
func decode(r *protocol.Reader) (Message, error) {
	var m Message
	var err error
	var localID uint64
	if localID, err = r.Uint64(); err != nil {
		return m, err
	}
	m.LocalEntityID = entity.ID(localID)
	if m.RegionX, err = r.Int32(); err != nil {
		return m, err
	}
	if m.RegionZ, err = r.Int32(); err != nil {
		return m, err
	}
	if m.Plane, err = r.Uint8(); err != nil {
		return m, err
	}
	count, err := r.Uint16()
	if err != nil {
		return m, err
	}
	m.Entities = make([]Snapshot, count)
	for i := range m.Entities {
		e := &m.Entities[i]
		var kind uint8
		var id uint64
		if id, err = r.Uint64(); err != nil {
			return m, err
		}
		e.ID = entity.ID(id)
		if kind, err = r.Uint8(); err != nil {
			return m, err
		}
		e.Kind = world.Kind(kind)
		if e.DefinitionID, err = r.Uint16(); err != nil {
			return m, err
		}
		if e.CharacterID, err = r.Uint64(); err != nil {
			return m, err
		}
		if e.Position.X, err = r.Int32(); err != nil {
			return m, err
		}
		if e.Position.Z, err = r.Int32(); err != nil {
			return m, err
		}
		if e.Position.Plane, err = r.Uint8(); err != nil {
			return m, err
		}
		if e.AppearanceID, err = r.Uint16(); err != nil {
			return m, err
		}
	}
	return m, nil
}
func encode(w *protocol.Writer, m Message) error {
	w.Uint64(uint64(m.LocalEntityID))
	w.Int32(m.RegionX)
	w.Int32(m.RegionZ)
	w.Uint8(m.Plane)
	w.Uint16(uint16(len(m.Entities)))
	for _, e := range m.Entities {
		w.Uint64(uint64(e.ID))
		w.Uint8(uint8(e.Kind))
		w.Uint16(e.DefinitionID)
		w.Uint64(e.CharacterID)
		w.Int32(e.Position.X)
		w.Int32(e.Position.Z)
		w.Uint8(e.Position.Plane)
		w.Uint16(e.AppearanceID)
	}
	return nil
}

func FromEntities(local entity.ID, regionX, regionZ int32, plane uint8, entities []world.RuntimeEntity) Message {
	m := Message{LocalEntityID: local, RegionX: regionX, RegionZ: regionZ, Plane: plane}
	for _, e := range entities {
		m.Entities = append(m.Entities, Snapshot{ID: e.ID, Position: e.Position, Kind: e.Kind, DefinitionID: e.DefinitionID, CharacterID: uint64(e.CharacterID), AppearanceID: e.AppearanceID})
	}
	return m
}
