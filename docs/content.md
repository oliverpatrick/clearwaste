# Content Architecture

## Purpose

`content_data` is the canonical authoring source for MMO game definitions and world data.

The content format should remain:

* human-readable during development
* engine independent
* usable by both server and client tooling
* based around stable numeric registry IDs
* suitable for compilation into compact runtime formats later

Godot is currently the client, but canonical content must not depend on Godot.

Do not place Godot paths such as:

```text
res://items/bronze_axe.tscn
```

inside shared content definitions.

Godot-specific asset bindings belong in the Godot client.

---

# Current layout

The current content layout is:

```text
content_data/
├── items/
│   ├── bronze_axe.json
│   ├── bronze_pickaxe.json
│   └── coin.json
│
├── map/
│   ├── region_0_0.json
│   ├── region_0_1.json
│   └── ...
│
├── mobs/
│   └── human/
│       ├── male.json
│       └── female.json
│
├── protocol/
│
└── schema/
    └── manifest.json
```

This structure may evolve, but changes should be incremental and should not introduce parallel competing content systems.

---

# Content ownership

Content falls into four broad categories.

## Server-only content

The server is authoritative for game rules.

Examples:

* combat formulas
* NPC AI
* aggression rules
* pathfinding rules
* drop tables
* respawn rules
* quest conditions
* quest rewards
* authoritative collision
* triggers
* hidden gameplay flags
* character persistence
* item effects

This information should not be sent to the client unless the client genuinely requires some presentation-safe subset.

---

## Shared content

Some definition information is required by both client and server.

Examples:

* item IDs
* NPC IDs
* object IDs
* animation IDs
* region IDs
* display names where required
* object size where required for rendering
* public interaction options
* public visual metadata

Shared content should use stable numeric IDs.

---

## Client-only content

Examples:

* textures
* models
* sprites
* sounds
* music
* shaders
* UI resources
* animation resources
* Godot scenes
* Godot materials
* asset registry bindings

These belong to the client and must not become part of the canonical MMO definition format.

---

## Runtime replicated state

Runtime world state is authoritative on the server and replicated to the client as required.

Examples:

* entity spawn
* entity despawn
* movement
* current animation
* visible equipment
* ground-item spawn/removal
* health changes
* combat effects
* chat
* inventory updates
* object state changes

Runtime state should not be confused with static content definitions.

---

# Registry IDs

Definitions should use stable numeric IDs.

Examples:

```text
Item ID      1
NPC ID       40
Object ID    102
Animation ID 7
```

Runtime systems should reference these IDs rather than:

* filenames
* display names
* JSON paths
* Godot paths
* arbitrary strings

For example:

```json
{
  "id": 1,
  "name": "Bronze axe"
}
```

A map placement should then reference:

```json
{
  "itemId": 1
}
```

rather than:

```json
{
  "item": "bronze_axe.json"
}
```

or:

```json
{
  "item": "Bronze axe"
}
```

Names are presentation/search values.

IDs are identity.

---

# ID stability

Once an ID has been used in persistent saves, maps or the network protocol, it should be treated as stable.

Do not casually reuse removed IDs.

For example, if:

```text
Item 37 = Bronze axe
```

is removed, do not later make:

```text
Item 37 = Dragon sword
```

Existing saves or map data may still refer to 37.

Prefer leaving retired IDs unused.

---

# Item definitions

Item source definitions currently live under:

```text
content_data/items/
```

For example:

```text
bronze_axe.json
bronze_pickaxe.json
coin.json
```

A minimal conceptual item definition may contain:

```json
{
  "id": 1,
  "name": "Bronze axe",
  "description": "A basic bronze axe."
}
```

Additional gameplay properties can be added as required.

Do not add speculative fields before a feature needs them.

Potential future server-facing properties include:

* stackability
* tradeability
* equipment slot
* equipment stats
* tool type
* value
* weight
* actions

Potential client-facing properties include:

* display name
* examine text
* visual ID
* icon ID

The source definition may contain both server/client-safe information initially, but later compilation may produce separate client and server runtime artifacts.

---

# Mob/NPC definitions

Mob definitions currently live under:

```text
content_data/mobs/
```

For example:

```text
content_data/mobs/human/male.json
content_data/mobs/human/female.json
```

Mob definitions should have stable numeric IDs just like items.

Conceptually:

```json
{
  "id": 100,
  "name": "Man",
  "description": "A regular person."
}
```

Server-only additions may include:

* stats
* aggression
* attack speed
* wander radius
* chase distance
* drops
* respawn timing
* behaviour/script references

Client-facing additions may include:

* visual ID
* animation set ID
* size
* public interaction actions

Do not embed Godot asset paths in these definitions.

---

# World and regions

The game world is logically 2D.

World geography is split into streamed regions/chunks.

Current files resemble:

```text
region_0_0.json
region_0_1.json
region_1_0.json
...
```

A region should describe static world content associated with that region.

Potential region information includes:

* region coordinates
* size
* planes
* terrain
* height
* collision
* edge/wall collision
* objects
* NPC spawns
* ground-item spawns
* zones
* triggers

The exact representation must evolve from the existing schema rather than replacing it unnecessarily.

---

# Coordinates

A region should have one documented coordinate convention.

Do not mix local tile coordinates and absolute world coordinates without explicitly naming them.

A preferred model is:

```text
region coordinate:
    regionX
    regionY

placement coordinate:
    localX
    localY
    plane
```

Then world coordinates can be calculated from the region size.

For a 64x64 region:

```text
worldX = regionX * 64 + localX
worldY = regionY * 64 + localY
```

If the existing map format already uses absolute world coordinates, preserve that convention until intentionally migrated.

Codex must inspect current region files before changing coordinate semantics.

---

# Planes

If maps support multiple planes, plane identity should be explicit.

For example:

```json
{
  "plane": 0,
  "x": 12,
  "y": 31
}
```

Avoid inferring plane from unrelated structure when the data crosses system boundaries.

Typical valid planes may be:

```text
0
1
2
3
```

but validation should follow the actual world format.

---

# Ground item placements

Regions need to support initial ground-item placements.

A ground item placement represents an item that exists at a particular world tile when the world content is initially loaded.

Conceptually:

```json
{
  "groundItems": [
    {
      "itemId": 1,
      "x": 17,
      "y": 31,
      "plane": 0,
      "quantity": 1
    }
  ]
}
```

Exact field names should follow the existing region conventions.

Required concepts are:

* item definition ID
* position
* quantity

Quantity should normally be greater than zero.

Item IDs must reference existing item definitions.

---

# Static definition vs runtime state

A region ground-item entry is an initial world definition.

It does not mean the client can assume the item permanently exists.

The intended flow is:

```text
region JSON
    ↓
server world loader
    ↓
authoritative runtime ground-item state
    ↓
visibility system
    ↓
network replication
    ↓
client
```

If a player picks the item up, the server changes authoritative runtime state.

The original JSON does not immediately change.

The item may later:

* respawn
* stay removed
* be replaced
* be dynamically created

depending on game rules.

---

# Client map streaming

The Godot client may stream static terrain and presentation data by region.

However, dynamic/interactable entities should ultimately come from authoritative server replication.

For example:

```text
terrain
    may be loaded from client content

ground item currently present
    comes from server state

NPC currently present
    comes from server state

player
    comes from server state
```

This prevents the client from rendering stale world state purely because an entity exists in static map JSON.

---

# Objects

Future region object placements should follow the same registry-ID model.

Conceptually:

```json
{
  "objects": [
    {
      "objectId": 50,
      "x": 20,
      "y": 12,
      "plane": 0,
      "rotation": 1
    }
  ]
}
```

Objects may include things such as:

* trees
* rocks
* doors
* scenery
* interactable world objects

Definition identity and placement identity should remain separate.

---

# NPC spawns

NPC placement should reference an NPC definition ID.

Conceptually:

```json
{
  "npcs": [
    {
      "npcId": 100,
      "x": 30,
      "y": 42,
      "plane": 0
    }
  ]
}
```

The region contains spawn configuration.

The actual NPC is created as a runtime ECS entity by the server.

---

# Collision

Collision is authoritative on the server.

The client may contain matching collision information for:

* local path preview
* cursor feedback
* presentation
* camera behaviour

but client collision must never override server validation.

The world uses tile-based movement with edge/outline blocking.

Collision between adjacent tiles should be represented explicitly rather than treating every blocked tile as fully impassable where directional edge blocking is required.

---

# Terrain and height

Terrain definitions should remain engine independent.

Possible source information includes:

* terrain/material ID
* colour
* rotation
* height
* overrides
* blend metadata

The client is responsible for turning terrain definitions into Godot geometry/rendering.

Godot-specific mesh or scene representations should not appear in canonical region JSON.

---

# Client asset registry

The Godot client maintains its own binding between stable content IDs and Godot presentation assets.

Conceptually:

```text
Item ID 1
    ↓
Godot ItemAssetRegistry
    ↓
bronze axe icon/model/scene
```

For example, the Godot registry may internally map:

```text
1 → res://assets/items/bronze_axe/...
```

That mapping is Godot-specific and should remain inside the client.

The server never needs to know this path.

---

# Source vs runtime format

JSON is currently the authoring format.

That does not mean JSON must remain the production runtime format.

Long term:

```text
content_data JSON
        ↓
content compiler
        ↓
    ┌───┴────┐
    ▼        ▼
server      client
binary      binary/assets
content     content
```

Possible compiled data includes:

* item registry
* NPC registry
* object registry
* map regions
* collision
* client-visible definitions

Do not build this compiler until runtime needs justify it.

The current priority is correct schemas and stable definitions.

---

# Schemas

Schemas live under:

```text
content_data/schema/
```

Schemas should validate authoring errors early.

Useful validation includes:

* required IDs
* valid numeric ranges
* nonzero quantities
* valid rotations
* valid planes
* duplicate registry IDs
* unknown definition references
* malformed map coordinates

Schema validation alone may not be sufficient for cross-file references.

For example:

```text
region says itemId = 37
```

requires registry/content validation to verify that item 37 actually exists.

---

# Content loading

Server content loading should conceptually occur in stages:

```text
load definitions
    ↓
validate IDs
    ↓
build registries
    ↓
load maps
    ↓
validate references against registries
    ↓
build world
```

This means maps are loaded after referenced definition registries are available.

Do not silently accept broken content references.

A bad map reference should fail content validation clearly.

---

# Content design rule

Prefer:

```text
small explicit definitions
stable IDs
schema validation
incremental extensions
```

over:

```text
large speculative schemas
implicit conventions
engine-specific paths
string identity
duplicated definitions
```

The source content should describe the game.

The server determines authoritative behaviour.

The client determines presentation.
