# MMO Architecture

## Overview

The project is an MMO with a Go authoritative backend and a Godot client.

The design should preserve the ability to replace Godot in the future. Shared game content, protocol definitions, map data and IDs must therefore remain engine independent.

The world is logically 2D and uses streamed regions/chunks.

The server uses ECS for runtime world entities and a custom binary protocol for transport.

## High-level flow

The intended application flow is:

1. User authenticates.

   * Username/password.
   * Google OAuth.
   * Apple OAuth.
   * Other identity providers may be added later.

2. Account is resolved.

3. User selects one of the characters owned by the account.

4. User selects a world.

5. A short-lived world login ticket identifies:

   * Account ID.
   * Character ID.
   * World ID.

6. Client connects to the selected world server.

7. World server:

   * validates the ticket
   * verifies capacity
   * acquires exclusive ownership of the character session
   * loads persistent character state
   * creates a runtime ECS player entity
   * inserts it into the simulation
   * calculates initial visibility
   * sends initial replicated state to the client

## Identity model

Keep these IDs distinct.

### Account ID

Permanent identity of the user/account.

An account can own multiple characters.

Example:

Account:

* Test

Characters:

* test_char1
* test_char2
* test_char3

### Character ID

Permanent identity of one playable character.

Character names are display/search data and must not be used as permanent internal identity.

### Entity ID

Temporary runtime identity of an entity in a particular world process.

A character can receive a different Entity ID each time it logs in.

### World ID

Identifies a selectable world/game-server instance.

### Connection ID

Temporary network connection identity.

These types should not be treated as interchangeable.

## Processes and scaling

Do not begin with a large microservice architecture.

The intended architecture is a modular monolith/monorepo with strong domain boundaries.

Initially:

* Account process.
* One or more world-server processes.

The same world-server binary can run multiple times using different configuration.

Example:

* World 1: up to 2,000 players.
* World 2: up to 2,000 players.
* World 3: up to 2,000 players.

Each world owns its own:

* ECS world.
* simulation loop.
* spatial index.
* NPC runtime state.
* connected players.

Additional services should only be split into independently deployed services when scaling, fault isolation or deployment requirements justify it.

Potential domains include:

* account
* character
* world
* game
* social
* leaderboard

## Account domain

Responsible for:

* login identities
* authentication
* account sessions
* OAuth identity mappings
* account-level state

Authentication provider details must not leak into game simulation code.

Downstream systems should generally work with Account IDs.

## Character domain

A character is persistent even when offline.

Responsible for:

* character ownership
* character creation
* character selection
* character metadata
* character session/online lock
* loading and saving persistent character state

A Player is a Character currently instantiated in a world.

## Character persistence

Persistent character gameplay state may be stored in versioned `.sav` data.

Examples include:

* position
* inventory
* equipment
* bank
* skills
* quest progression
* appearance
* unlocks
* gameplay variables

Searchable/control-plane metadata should be stored in the database where appropriate.

Examples:

* character ID
* account ID
* character name
* creation time
* last login
* save version/reference

Do not serialize raw ECS implementation details directly as the persistent format.

Persistent save formats should be versioned and migratable.

When worlds eventually run on multiple physical machines, save storage must be accessible across those machines through an appropriate repository/storage implementation.

## Membership

Membership/subscription state should live at the level that actually owns the concept.

If membership applies to every character owned by an account, it is account-level state.

Do not duplicate the same authoritative membership flag across account and character tables.

## World domain

A World is a selectable running game-server instance.

World information can include:

* ID
* display name
* status
* player count
* capacity
* connection endpoint

The world itself is authoritative for its actual capacity.

The world-selection service may prevent obviously full worlds being selected, but the world must verify capacity again when the connection arrives.

## Simulation

The ECS spatial grid is not the concurrency boundary.

A tick is separated into phases.

### Decide

* parallel across entities/work
* read-only snapshot
* no mutation of authoritative world state
* AI/pathfinding/target selection occurs here

### Apply

* processes generated intents
* mutates authoritative state
* resolves conflicts
* may temporarily group mutations spatially

Chunks do not permanently own entities.

### Spatial rebuild

Rebuild the proximity index using new authoritative positions.

The index is primarily used for:

* nearby-entity queries
* visibility
* viewport replication
* other spatial lookups

## Content architecture

`content_data` contains canonical authoring definitions.

Current structure:

content_data/

* items/

  * bronze_axe.json
  * bronze_pickaxe.json
  * coin.json
* map/

  * region_0_0.json
  * region_0_1.json
  * ...
* mobs/

  * human/

    * male.json
    * female.json
* protocol/
* schema/

  * manifest.json

The authoring format is JSON for readability.

Runtime client/server formats may later be compiled into more compact binary representations.

Do not tie canonical definitions to Godot.

## Registry IDs

Items, NPCs/mobs, objects and other definitions should have stable registry IDs.

Runtime systems and map placements should reference registry IDs rather than filenames or display names.

Example:

Item definition:

* ID 100
* name: Bronze axe

Map placement:

* item ID: 100
* x: 17
* y: 31
* quantity: 1

The Godot client separately binds client-facing definition or visual IDs to actual Godot assets.

## Map and chunk content

Map regions are canonical descriptions of world geography/content.

Regions may eventually include:

* terrain
* height
* tile data
* outline/edge collision
* objects
* NPC spawns
* ground item spawns
* triggers
* zones

For the current implementation, keep additions incremental.

Ground-item placements should be stored in the appropriate region so both world loading and client presentation can associate the item with the streamed chunk.

A reasonable conceptual structure is:

```json
{
  "regionX": 0,
  "regionY": 0,
  "groundItems": [
    {
      "itemId": 100,
      "x": 17,
      "y": 31,
      "quantity": 1
    }
  ]
}
```

Exact field names and coordinate semantics must be reconciled with the existing region JSON/schema before implementation.

Static placement in content does not make the client authoritative.

The server loads authoritative world state from the definition.

The client receives or renders the appropriate visible runtime state according to the replication design.

## Client architecture

Godot is currently used for:

* rendering
* input
* UI
* animation
* sound
* client-side interpolation/presentation

It is not authoritative for gameplay.

Godot-specific assets should be mapped through a client registry.

For example:

NPC definition ID 42
→ client registry
→ Godot model/scene/animation resources

Canonical content must not contain `res://` resource paths.

## Content split

### Server-only

Examples:

* combat calculations
* AI
* hidden NPC behaviour
* drop tables
* authoritative collision/rules
* quest conditions
* persistent character state

### Shared

Examples:

* stable definition IDs
* map/region identity
* item/NPC/object identity
* client-visible definition metadata when required

### Client-only

Examples:

* textures
* models
* sprites
* animation assets
* sound
* UI resources
* shaders
* Godot asset bindings

### Runtime replicated state

Examples:

* entity spawn/despawn
* positions
* movement
* visible equipment/appearance
* animations
* ground-item appearance/removal
* inventory updates for the local player
* chat
* combat presentation state

## Design principle

The server decides what is true.

The client decides how that truth is presented.
