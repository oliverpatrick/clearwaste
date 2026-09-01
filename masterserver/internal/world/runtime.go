package world

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"master/clearwaste/internal/character"
	"master/clearwaste/internal/game/entity"
	"master/clearwaste/internal/game/movement"
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
	mu       sync.Mutex
	next     entity.ID
	Entities []RuntimeEntity
	pending  map[entity.ID][]movement.Direction
	blocked  map[[3]int32]bool
	edges    map[[4]int32]bool
}

func NewState(root string) (*State, error) {
	s := &State{next: 1, pending: map[entity.ID][]movement.Direction{}, blocked: map[[3]int32]bool{}, edges: map[[4]int32]bool{}}
	if err := s.loadRegion(filepath.Join(root, "map", "region_0_0.json")); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *State) Add(e RuntimeEntity) entity.ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.add(e)
}

func (s *State) add(e RuntimeEntity) entity.ID {
	e.ID = s.next
	s.next++
	s.Entities = append(s.Entities, e)
	return e.ID
}

func (s *State) QueueStep(id entity.ID, direction movement.Direction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending[id]) < 32 {
		s.pending[id] = append(s.pending[id], direction)
	}
}

func (s *State) Tick() []RuntimeEntity {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := []RuntimeEntity{}
	for id, steps := range s.pending {
		if len(steps) == 0 {
			continue
		}
		var current *RuntimeEntity
		for i := range s.Entities {
			if s.Entities[i].ID == id {
				current = &s.Entities[i]
				break
			}
		}
		if current == nil {
			continue
		}
		dx, dz := movement.Delta(steps[0])
		if s.walkable(current.Position, dx, dz) {
			current.Position.X += dx
			current.Position.Z += dz
			changed = append(changed, *current)
		}
		s.pending[id] = steps[1:]
	}
	return changed
}

func (s *State) walkable(p Position, dx, dz int32) bool {
	if s.blocked[[3]int32{p.X + dx, p.Z + dz, int32(p.Plane)}] {
		return false
	}
	if dx != 0 && dz != 0 && (!s.walkable(p, dx, 0) || !s.walkable(p, 0, dz)) {
		return false
	}
	if dx != 0 && s.edges[[4]int32{p.X, p.Z, int32(p.Plane), dx}] {
		return false
	}
	if dz != 0 && s.edges[[4]int32{p.X, p.Z, int32(p.Plane), dz}] {
		return false
	}
	return true
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
			Plane     uint8
			Collision struct {
				BlockedTiles [][]int32 `json:"blockedTiles"`
				Walls        []struct {
					X, Y int32
					Edge string
				} `json:"walls"`
			} `json:"collision"`
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
		for _, tile := range p.Collision.BlockedTiles {
			if len(tile) >= 2 {
				s.blocked[[3]int32{tile[0] + src.RegionX*64, tile[1] + src.RegionY*64, int32(p.Plane)}] = true
			}
		}
		for _, wall := range p.Collision.Walls {
			var d int32
			switch wall.Edge {
			case "east":
				d = 1
			case "west":
				d = -1
			case "south":
				d = 2
			case "north":
				d = -2
			}
			if d != 0 {
				s.edges[[4]int32{wall.X + src.RegionX*64, wall.Y + src.RegionY*64, int32(p.Plane), d}] = true
			}
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
