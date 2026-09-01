package world

import (
	"encoding/json"
	"os"
	"path/filepath"

	"master/clearwaste/internal/character"
	"master/clearwaste/internal/game/entity"
)

type Kind uint8

const (
	KindPlayer Kind = iota
	KindNPC
	KindObject
	KindGroundItem
)

type Position struct {
	X, Z  int32
	Plane uint8
}
type RuntimeEntity struct {
	ID           entity.ID
	Position     Position
	Kind         Kind
	DefinitionID uint16
	CharacterID  character.ID
	AppearanceID uint16
}

type Character struct {
	ID           character.ID
	Name         string
	AppearanceID uint16
	Spawn        Position
}

type State struct {
	next     entity.ID
	Entities []RuntimeEntity
}

func NewState(root string) (*State, error) {
	s := &State{next: 1}
	if err := s.loadRegion(filepath.Join(root, "map", "region_0_0.json")); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) Add(e RuntimeEntity) entity.ID {
	e.ID = s.next
	s.next++
	s.Entities = append(s.Entities, e)
	return e.ID
}

func DevelopmentCharacter(id character.ID) (Character, bool) {
	if id != 1 {
		return Character{}, false
	}
	return Character{ID: 1, Name: "Development", AppearanceID: 0, Spawn: Position{X: 10, Z: 10, Plane: 0}}, true
}

func (s *State) SpawnPlayer(c Character) entity.ID {
	return s.Add(RuntimeEntity{Position: c.Spawn, Kind: KindPlayer, CharacterID: c.ID, AppearanceID: c.AppearanceID})
}

func (s *State) Visible(regionX, regionZ int32, plane uint8) []RuntimeEntity {
	result := make([]RuntimeEntity, 0, len(s.Entities))
	for _, e := range s.Entities {
		if e.Position.Plane == plane && e.Position.X/64 == regionX && e.Position.Z/64 == regionZ {
			result = append(result, e)
		}
	}
	return result
}

func (s *State) loadRegion(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var src struct {
		RegionX, RegionY int32
		Planes           []struct {
			Plane       uint8
			GameObjects []struct {
				ObjectID uint16
				X, Y     int32
			}
			MobSpawns []struct {
				NPCID uint16
				X, Y  int32
			}
			Ground []struct {
				ItemID   uint16
				X, Y     int32
				Quantity uint32
			} `json:"groundItemSpawns"`
		}
	}
	if err := json.Unmarshal(b, &src); err != nil {
		return err
	}
	for _, p := range src.Planes {
		if p.Plane != 0 {
			continue
		}
		for _, o := range p.GameObjects {
			s.Add(RuntimeEntity{Position: Position{o.X + src.RegionX*64, o.Y + src.RegionY*64, p.Plane}, Kind: KindObject, DefinitionID: o.ObjectID})
		}
		for _, n := range p.MobSpawns {
			s.Add(RuntimeEntity{Position: Position{n.X + src.RegionX*64, n.Y + src.RegionY*64, p.Plane}, Kind: KindNPC, DefinitionID: n.NPCID})
		}
		for _, g := range p.Ground {
			s.Add(RuntimeEntity{Position: Position{g.X + src.RegionX*64, g.Y + src.RegionY*64, p.Plane}, Kind: KindGroundItem, DefinitionID: g.ItemID})
		}
	}
	return nil
}
