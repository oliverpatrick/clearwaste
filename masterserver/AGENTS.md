# AGENTS.md

## Purpose

This repository contains a Go MMO server built around:

- an ECS for runtime entity/component state;
- a logically 2D world;
- a fixed-tick simulation;
- an immutable-read / intent / apply simulation model;
- a spatial index used for proximity queries, not entity ownership;
- a custom binary network protocol;
- content definitions loaded/compiled into runtime world state;
- one game-server process per selectable world.

This file is the primary instruction set for AI coding agents working in the repository.

If a task conflicts with an explicit architectural decision documented here or under `docs/`, do not silently invent a new pattern. Prefer the existing boundary. If the task genuinely requires changing a boundary, make that change explicit and update the relevant documentation.

## Read these before making architectural changes

1. `docs/architecture.md`
2. `docs/folder-structure.md`
3. `docs/simulation.md`
4. `docs/network-protocol.md`
5. `docs/identity-persistence.md`
6. `docs/testing-conventions.md`
7. `docs/mvp-scope.md`

## Architectural rules

### 1. `cmd/` is composition only

Executable entry points belong under `cmd/`.

Current intended executables:

```text
cmd/
├── account/
└── server/
```

`cmd/*/main.go` should:

- load configuration;
- construct dependencies;
- start the application;
- handle shutdown.

It should not contain gameplay rules, SQL queries, packet parsing logic, ECS systems, or business logic.

Application wiring belongs under `internal/app/`.

### 2. Keep engine infrastructure separate from game rules

`internal/engine/` is reusable runtime infrastructure.

Examples:

- ECS storage and queries;
- simulation scheduling;
- worker pools;
- spatial indexing;
- transport/framing;
- scripting runtime.

`internal/game/` contains MMO gameplay rules.

Examples:

- movement;
- combat;
- inventory;
- actions;
- animations;
- NPC behaviour;
- visibility;
- skills;
- quests.

The engine controls **how and when** work runs. Game packages control **what the rules mean**.

### 3. The spatial grid is not a concurrency boundary

Do not make chunks/cells own entities or long-lived goroutines.

The spatial index exists to answer questions such as:

- what is at this tile?
- what entities are near this entity?
- what belongs in a player's viewport?

The simulation is sharded by work/entities, not by permanent spatial ownership.

### 4. Tick phases are explicit

The core tick model is:

```text
snapshot
   ↓
DECIDE      read-only, parallelisable
   ↓ barrier
APPLY       authoritative mutation
   ↓ barrier
REBUILD     refresh spatial read-side index
   ↓
VISIBILITY / OUTBOUND UPDATES
```

During `Decide`:

- systems read immutable snapshot data;
- systems produce typed intents;
- systems must not mutate authoritative ECS state.

During `Apply`:

- intents are validated/resolved;
- positions, health, inventory, equipment, etc. may mutate;
- conflict resolution must be deterministic.

During `Rebuild`:

- rebuild the spatial index from authoritative post-apply positions;
- prefer reusable buffers over per-tick allocation.

Never mutate ECS state directly from network reader goroutines.

### 5. Runtime identity types must be distinct

Do not use plain `uint64` interchangeably for unrelated IDs.

Prefer explicit types such as:

```go
type AccountID uint64
type CharacterID uint64
type EntityID uint64
type WorldID uint16
type ItemID uint32
type NPCID uint32
type ObjectID uint32
```

Important distinctions:

- `AccountID`: login/customer identity;
- `CharacterID`: persistent playable identity;
- `EntityID`: runtime ECS identity;
- `WorldID`: selectable game-server/world identity;
- content IDs: stable numeric definition IDs.

Character names and content names are display/content-authoring values, not runtime identity.

### 6. One account may own multiple characters

The model is:

```text
Account
  ├── Character
  ├── Character
  └── Character
```

A selected character is loaded into one world and instantiated as an ECS entity.

Do not make an Account the entity that walks around the world.

### 7. One `cmd/server` process represents one world

The same server binary may run multiple times with different world configuration.

Example:

```text
World 1 -> server process A
World 2 -> server process B
World 3 -> server process C
```

Each process owns its own:

- ECS;
- simulation runner;
- spatial index;
- NPC state;
- connected players;
- tick loop.

The planned world capacity is 2,000 players per world.

Do not distribute one world's ECS across machines unless a future requirement explicitly demands it.

### 8. Prefer a modular monolith over premature microservices

Keep domain boundaries clean, but use in-process Go calls where practical.

Potential domains include:

```text
account/
character/
world/
game/
social/
leaderboard/
content/
```

A package does not need to become a network service simply because it is a separate domain.

Extract a service only when it needs independent scaling, deployment, failure isolation, ownership, or operational behaviour.

### 9. Network code stops at typed gameplay input/output

The intended inbound flow is:

```text
socket bytes
   ↓
frame decoder
   ↓
opcode codec
   ↓
typed request
   ↓
bounded inbound queue
   ↓
simulation/game system
```

Packet handlers must not directly mutate arbitrary ECS state.

The intended outbound flow is:

```text
game/visibility or gameplay result
   ↓
typed outbound message
   ↓
opcode codec
   ↓
bounded connection send queue
   ↓
writer goroutine
```

### 10. Persistence is not a raw ECS dump

Character saves should use an explicit versioned persistence model.

Do not serialize ECS storage/archetypes directly.

The persistence boundary is conceptually:

```text
ECS/runtime components
      ↓ extract
CharacterSave
      ↓ encode
versioned .sav
```

and on load:

```text
versioned .sav
      ↓ decode/migrate
CharacterSave
      ↓ instantiate
ECS/runtime components
```

The database stores queryable/control-plane metadata. `.sav` files store heavier gameplay state.

### 11. Content authoring and runtime content are separate concerns

Human-editable content may use JSON or another readable format.

Runtime server/client data may be compiled to a compact binary representation.

Do not optimise authoring files by making them unreadable.

Do not use content string names as hot-path identity if a stable numeric ID exists.

### 12. Optimise only after correctness boundaries exist

Priority order:

```text
correct
→ deterministic
→ tested
→ measured
→ parallelised/optimised
```

Avoid adding concurrency to compensate for an unclear state-ownership model.

## Current movement rules

The logical world is tile-based and supports 8-direction movement using Chebyshev adjacency.

Directions:

```text
NW  N  NE
 W  P  E
SW  S  SE
```

A one-step move satisfies:

```text
abs(dx) <= 1
abs(dy) <= 1
max(abs(dx), abs(dy)) == 1
```

Collision is represented by cardinal tile-edge blockers:

- North;
- East;
- South;
- West.

Diagonal movement must not cut blocked corners. A diagonal step requires both relevant cardinal traversals to be open.

Example: North-East requires both North and East to be traversable.

Keep opposite edges consistent. If tile A blocks East, the adjacent tile B must behave as blocked from the West.

## Protocol invariants

Current transport conventions:

- big-endian;
- frame header: `uint16 opcode` + `uint32 payload length`;
- one reader goroutine per connection;
- one writer goroutine per connection;
- bounded inbound/outbound queues;
- payload size validated before allocation;
- `io.ReadFull` semantics for fixed frame reads;
- unknown opcode is a protocol violation and closes the connection;
- malformed payload and unknown opcode are distinct errors;
- opcode values are explicit and stable once assigned.

Do not renumber an existing opcode just to keep an enum tidy.

## Testing expectations

Every behaviour change should normally include tests at the narrowest useful level.

Prefer:

1. pure unit tests for rules;
2. system/package tests for ECS/game interactions;
3. tick integration tests for phase behaviour;
4. protocol round-trip tests for codecs;
5. race testing for concurrency changes.

Important simulation tests should assert:

- state does not mutate during Decide;
- Apply produces deterministic outcomes;
- blocked movement stays blocked;
- diagonal corner cutting is rejected;
- spatial index reflects post-Apply state;
- network input cannot bypass simulation rules.

Run at minimum:

```bash
go test ./...
go vet ./...
```

For concurrency changes:

```bash
go test -race ./...
```

## Agent workflow

Before editing:

1. inspect the relevant package and neighbouring packages;
2. identify which domain owns the behaviour;
3. identify whether the task changes protocol, persistence, simulation, or content contracts;
4. preserve existing public APIs unless the task requires changing them;
5. write/update tests alongside the implementation.

While editing:

- keep files focused;
- prefer explicit typed structs over `map[string]any`;
- prefer typed IDs over strings for runtime references;
- avoid global mutable state;
- do not add a new dependency without a concrete need;
- reuse per-tick buffers where the code already has a buffer lifecycle;
- keep network I/O outside authoritative simulation mutation.

After editing:

1. run focused tests;
2. run `go test ./...`;
3. run `go vet ./...`;
4. run the race detector when concurrency changed;
5. update docs when an architectural or wire-format contract changed.

## Do not do these without an explicit architecture change

- Do not turn each chunk into a goroutine.
- Do not use the spatial grid as entity ownership.
- Do not mutate ECS state from the socket reader.
- Do not deep-copy the entire world every tick without evidence it is necessary.
- Do not dump ECS internals directly to `.sav`.
- Do not identify characters by name internally.
- Do not identify content by string name in hot runtime paths when numeric IDs exist.
- Do not split a package into a microservice merely because it has grown.
- Do not add Redis/PostgreSQL/OAuth to an MVP task unless the task is specifically about them.
- Do not implement speculative abstractions for features that do not yet exist.

## Design bias

When multiple implementations are valid, prefer the one that is:

- deterministic;
- explicit;
- testable without a live network;
- isolated behind a small interface;
- cheap to replace later;
- allocation-conscious in the tick path;
- simple enough to understand from one package at a time.
