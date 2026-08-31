# Server Architecture

## Overview

The server is designed as a modular Go codebase with multiple deployable processes, not as a network of mandatory microservices.

At runtime the main shape is:

```text
                         ┌────────────────┐
                         │     Client     │
                         └───────┬────────┘
                                 │
                         login / account
                                 │
                                 ▼
                     ┌─────────────────────┐
                     │   Account Process   │
                     │ auth / characters   │
                     │ world directory     │
                     └──────────┬──────────┘
                                │
                      character + world
                           selection
                                │
                        short-lived ticket
                                │
                                ▼
        ┌─────────────────────────────────────────────┐
        │             Game World Process              │
        │                                             │
        │ ECS → simulation → spatial → visibility     │
        │                                             │
        │ one process = one selectable world          │
        └─────────────────────────────────────────────┘
```

A world is planned to support at most 2,000 connected players.

Multiple worlds scale horizontally by running the same `cmd/server` binary with different configuration.

## Code architecture vs deployment architecture

The repository should retain clean domain packages even when several domains run in the same process.

```text
code:
  account/
  character/
  world/
  social/
  leaderboard/
  game/

deployment initially:
  account process
  world process 1
  world process 2
  ...
```

This avoids RPC overhead and operational complexity while preserving extraction boundaries if a subsystem later needs independent deployment.

## Layers

### `cmd/`

Process entry points.

Responsibilities:

- load configuration;
- construct application;
- start/stop process.

No domain logic.

### `internal/app/`

Application composition and lifecycle.

Examples:

- `internal/app/gameserver`;
- `internal/app/accountserver`.

Responsibilities:

- dependency wiring;
- startup order;
- graceful shutdown;
- application-level orchestration.

### `internal/engine/`

Runtime infrastructure independent of particular MMO rules.

Key packages:

```text
engine/
├── ecs/
├── simulation/
├── spatial/
├── network/
└── scripting/
```

### `internal/game/`

Runtime gameplay rules.

Examples:

```text
game/
├── player/
├── movement/
├── action/
├── animation/
├── item/
├── inventory/
├── combat/
├── npc/
├── visibility/
├── skills/
└── quests/
```

### Account and character domains

`account/` answers "who can authenticate?"

`character/` answers "which playable identities belong to that account?"

`game/player/` is the runtime representation of one loaded character in a world.

### `world/`

Control-plane representation of selectable worlds.

Responsibilities may include:

- world IDs;
- online/offline status;
- population/capacity;
- endpoint information;
- entry tickets.

This is distinct from `engine/spatial/`.

### `social/`

Cross-character social state such as:

- friends;
- ignore lists;
- presence;
- clans.

These relationships should normally use `CharacterID`, not `EntityID`.

### `leaderboard/`

Derived/global rankings keyed by persistent character identity.

### `content/`

Content definition loading, validation, registries, and compiled-content concerns.

## Dependency direction

Aim for dependencies that point inward toward small contracts rather than across implementation details.

A useful mental model:

```text
cmd
 ↓
app
 ↓
domain/game ──────→ engine abstractions
 ↓
repositories/interfaces
 ↓
storage adapters
```

Avoid:

- gameplay packages importing `cmd`;
- engine packages importing high-level game rules;
- network codecs reaching directly into storage;
- combat issuing SQL;
- persistence importing a concrete client presentation type.

## Runtime world lifecycle

A world server should load world content before it accepts players:

```text
process starts
  ↓
load configuration
  ↓
load/validate content definitions
  ↓
construct ECS
  ↓
construct collision/spatial structures
  ↓
spawn static objects/NPCs
  ↓
initial spatial build
  ↓
mark world ready
  ↓
accept players
```

Player entry should load the player into an already-running world rather than rebuilding world content per login.

## Character entry flow

Target flow:

```text
authenticate account
  ↓
select character
  ↓
select world
  ↓
issue world ticket
  ↓
connect to world
  ↓
validate ticket
  ↓
verify character ownership
  ↓
acquire character session/online lock
  ↓
load character save
  ↓
create ECS entity
  ↓
attach runtime components
  ↓
rebuild/update spatial state
  ↓
send initial viewport
```

## Scaling strategy

Scale the simulation by adding worlds before attempting to distribute a single world.

Example:

```text
host A:
  World 1
  World 2

host B:
  World 3
  World 4
```

If cross-world presence or messaging is later required, add a shared coordination mechanism without changing world ECS ownership.

Potential future infrastructure:

- PostgreSQL for queryable persistent metadata;
- Redis or similar for sessions, world tickets, online locks/presence;
- object/shared storage for `.sav` files;
- pub/sub for cross-world messaging.

These are future adapters, not reasons to redesign gameplay packages now.

## Service extraction rule

A module should become its own service only when there is a real requirement for one or more of:

- independent scaling;
- independent deployments;
- stronger failure isolation;
- distinct security boundary;
- distinct data ownership;
- independently operated team.

Code size alone is not enough.
