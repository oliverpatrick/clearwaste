# Folder Structure

This is the target structure. Not every package must exist immediately.

Create packages only when current work needs them.

```text
.
├── AGENTS.md
├── cmd/
│   ├── account/
│   │   └── main.go
│   └── server/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── accountserver/
│   │   │   ├── app.go
│   │   │   └── bootstrap.go
│   │   └── gameserver/
│   │       ├── app.go
│   │       ├── bootstrap.go
│   │       └── run.go
│   │
│   ├── engine/
│   │   ├── ecs/
│   │   │   ├── entity.go
│   │   │   ├── world.go
│   │   │   ├── storage.go
│   │   │   └── query.go
│   │   │
│   │   ├── simulation/
│   │   │   ├── runner.go
│   │   │   ├── snapshot.go
│   │   │   ├── buffers.go
│   │   │   ├── workers.go
│   │   │   └── tick.go
│   │   │
│   │   ├── spatial/
│   │   │   ├── direction.go
│   │   │   ├── cell.go
│   │   │   ├── grid.go
│   │   │   ├── collision.go
│   │   │   ├── rebuild.go
│   │   │   └── query.go
│   │   │
│   │   ├── network/
│   │   │   ├── server.go
│   │   │   ├── connection.go
│   │   │   ├── session.go
│   │   │   └── protocol/
│   │   │       ├── frame.go
│   │   │       ├── opcode.go
│   │   │       ├── reader.go
│   │   │       ├── writer.go
│   │   │       └── registry.go
│   │   │
│   │   └── scripting/
│   │
│   ├── game/
│   │   ├── player/
│   │   ├── movement/
│   │   │   ├── component.go
│   │   │   ├── intent.go
│   │   │   ├── decide.go
│   │   │   └── apply.go
│   │   ├── action/
│   │   ├── animation/
│   │   ├── item/
│   │   ├── inventory/
│   │   ├── combat/
│   │   │   ├── component.go
│   │   │   ├── intent.go
│   │   │   ├── decide.go
│   │   │   ├── apply.go
│   │   │   └── damage.go
│   │   ├── npc/
│   │   ├── visibility/
│   │   ├── skills/
│   │   └── quests/
│   │
│   ├── account/
│   │   ├── account.go
│   │   ├── repository.go
│   │   ├── auth/
│   │   │   ├── service.go
│   │   │   ├── identity.go
│   │   │   └── provider/
│   │   │       ├── password.go
│   │   │       ├── google.go
│   │   │       └── apple.go
│   │   └── session/
│   │
│   ├── character/
│   │   ├── character.go
│   │   ├── repository.go
│   │   ├── creation/
│   │   ├── session/
│   │   └── save/
│   │       ├── save.go
│   │       ├── codec.go
│   │       ├── version.go
│   │       ├── migrator.go
│   │       └── repository.go
│   │
│   ├── world/
│   │   ├── world.go
│   │   ├── directory.go
│   │   ├── status.go
│   │   └── ticket.go
│   │
│   ├── social/
│   │   ├── friends/
│   │   ├── ignore/
│   │   ├── presence/
│   │   └── clans/
│   │
│   ├── leaderboard/
│   ├── content/
│   │   ├── definitions/
│   │   ├── registry/
│   │   ├── loader/
│   │   └── validation/
│   │
│   └── storage/
│       ├── postgres/
│       ├── redis/
│       └── saves/
│
├── content/
│   ├── items/
│   ├── npcs/
│   ├── objects/
│   ├── maps/
│   └── quests/
│
└── docs/
```

## Package ownership guide

### `engine/ecs`

Owns:

- entity allocation;
- component storage;
- ECS queries;
- runtime component access.

Does not own:

- combat formulas;
- pathfinding policy;
- packet opcodes;
- character persistence schema.

### `engine/simulation`

Owns:

- tick sequencing;
- phase barriers;
- snapshot lifecycle;
- worker scheduling;
- reusable per-tick buffers.

Does not own domain rules.

### `engine/spatial`

Owns:

- tile/cell indexing;
- collision edge representation;
- nearby/at queries;
- spatial rebuild.

Does not own entity lifetime or concurrency ownership.

### `engine/network`

Owns:

- sockets;
- connections;
- queues/backpressure;
- frame reading/writing;
- protocol registry.

Feature codecs may remain feature-owned if that keeps packet formats near the gameplay feature. The transport should depend on codec interfaces rather than game internals.

### `game/movement`

Owns:

- movement requests;
- movement intent;
- movement validation rules;
- application of accepted movement.

Uses spatial collision data but does not own the spatial grid.

### `game/action`

Owns generic runtime player/NPC action scheduling where a common action abstraction is useful.

Avoid forcing every gameplay mechanic through a generic action interface if typed feature requests remain clearer.

### `game/animation`

Owns server-authoritative animation state/events.

The server uses IDs. Client-specific animation asset names belong on the client.

### `game/inventory` and `game/item`

`item` owns definitions/IDs.

`inventory` owns runtime slots/stacks/operations.

Do not make inventory own item definitions.

### `game/combat`

Owns:

- attack requests/intents;
- target/range/cooldown validation;
- damage resolution;
- combat state.

Movement and spatial queries are dependencies, not duplicated logic.

### `game/visibility`

Owns interest management:

- which entities a player should know about;
- spawn/move/remove visibility decisions.

`engine/network` transports the resulting messages.

### `account`

Owns authentication/customer identity.

### `character`

Owns persistent playable identity, ownership, session locks, and save boundaries.

### `world`

Owns selectable world metadata and world-entry control-plane concepts.

### `content`

Owns authoring/runtime definition loading and registries.

The root `content/` directory stores actual data, not Go application logic.

## Naming guidance

Prefer domain nouns over generic buckets.

Good:

```text
game/movement
character/save
engine/spatial
```

Avoid growing catch-all packages such as:

```text
utils
helpers
common
models
managers
misc
```

If a helper has one clear owner, put it with that owner.
