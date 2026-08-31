# MMO Project Instructions

This repository contains an MMO consisting of:

* A Go backend/server.
* A Godot client.
* Engine-independent game/content definitions under `content_data`.
* A custom binary network protocol.
* A logically 2D, chunked world.
* ECS-based world simulation on the Go server.

## Sources of truth

Before making architectural changes, read:

* `docs/architecture.md`
* `docs/content.md`
* `docs/protocol.md`
* `docs/simulation.md`

Also inspect the existing implementation before proposing new abstractions.

Existing code and schemas take precedence over examples in documentation when they clearly represent a newer decision. If documentation and implementation disagree, report the discrepancy rather than silently inventing a third design.

## General architecture

Keep these concepts separate:

* Account: the human/login identity.
* Character: a persistent playable identity owned by an account.
* Player: a character currently instantiated in a world.
* Entity ID: temporary runtime ECS identity.
* World: one selectable game-server instance, with a target maximum of 2,000 players.
* Spatial index: used for proximity/querying, not as the concurrency ownership boundary.

Do not introduce microservices unless a ticket explicitly requires one.

The intended deployment model is a modular monolith/monorepo with separate processes where useful, particularly one process per game world.

## Server

The server is authoritative.

The client must not be trusted for:

* Position.
* Collision.
* Inventory state.
* Combat outcomes.
* Skills/XP.
* Quest progression.
* Item ownership.
* NPC behaviour.
* Drop tables.
* Character persistence.

Prefer feature/domain packages over generic `utils`, `managers`, or large handler packages.

Keep infrastructure and game rules separate.

## Simulation

The world simulation follows a phased tick model:

1. Decide.

   * Parallel.
   * Read-only snapshot.
   * Expensive work such as AI, pathfinding and target selection happens here.

2. Apply.

   * Mutates authoritative state.
   * Resolves conflicts.
   * Cheap operations may be partitioned spatially where useful.
   * Spatial chunks are not permanent concurrency owners.

3. Rebuild spatial index.

   * Rebuild proximity/query structures from authoritative positions.
   * Spatial data is a read-side index.

Visibility/replication is calculated from the resulting authoritative state.

Avoid per-tick allocation where practical. Prefer reusable/preallocated tick buffers.

## Content

`content_data` is the canonical authoring source for game definitions.

Current structure includes:

* `content_data/items`
* `content_data/map`
* `content_data/mobs`
* `content_data/protocol`
* `content_data/schema`

Definitions should use stable numeric registry IDs for runtime references.

Do not put Godot resource paths such as `res://...` into canonical shared content.

Godot binds definition/visual IDs to Godot-specific assets through its own client registry.

Content should remain portable to another client engine.

JSON is currently an authoring/readability format. Do not assume it must be the final runtime format. It may later be compiled into compact binary client/server artifacts.

## World/map data

The world is logically 2D and split into streamed regions/chunks.

Region definitions can contain placements for static/runtime-spawned content such as:

* Objects.
* NPC spawn definitions.
* Ground item placements.
* Terrain.
* Collision.
* Other world metadata.

Placements should refer to definitions by stable IDs, not file paths or display names.

For example, a ground item placement conceptually contains:

* item definition ID
* tile coordinates
* quantity

The exact JSON representation must follow or extend the existing region schema rather than introducing an unrelated parallel format.

## Client

Godot is the current client implementation but is not the canonical content format.

Keep these concerns separate:

* engine-independent definitions/data
* networking/protocol
* Godot presentation
* Godot asset bindings
* runtime replicated entity state

Do not reproduce the server ECS in Godot.

Godot nodes/scenes are presentations of replicated server entities.

The client may cache static content but receives authoritative dynamic state from the world server.

## Networking

The project uses a custom binary protocol.

Prefer stable numeric opcodes and IDs.

Keep protocol framing separate from gameplay behaviour.

Packets should communicate compact runtime identifiers and state, not Godot resource paths or large repeated definition strings.

Do not change existing wire formats casually. Treat opcode values and binary layouts as stable compatibility contracts once introduced.

## Development rules

For every ticket:

1. Inspect relevant existing code, schemas and tests first.
2. State any important architectural assumption before implementing it.
3. Make the smallest coherent change that completes the ticket.
4. Do not refactor unrelated code.
5. Add or update tests where appropriate.
6. Run relevant formatting, linting and test commands.
7. Report:

   * what changed
   * files changed
   * tests run
   * architectural decisions made
   * follow-up work intentionally left out

Prefer small incremental tickets over large feature rewrites.
