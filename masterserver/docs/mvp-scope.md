# MVP Scope and Near-Term Implementation Guidance

## Objective

The first playable server slice should prove the architecture, not implement the full MMO platform.

The initial vertical slice is:

```text
start server
  ↓
connect player
  ↓
spawn on small map
  ↓
move 8 directions
  ↓
respect N/E/S/W edge blockers
  ↓
inventory contains simple items
  ↓
perform typed action
  ↓
server emits animation
  ↓
attack a target
  ↓
target loses health
```

## In scope

### Simulation

- 600 ms tick;
- Snapshot → Decide → Apply → Rebuild;
- initially correct and deterministic before aggressive parallelisation.

### ECS

Minimum components needed for the vertical slice, likely including:

- Player;
- Position;
- Movement/input state;
- Inventory;
- Equipment;
- Health;
- Combat;
- Animation.

Do not build an abstract "perfect ECS" before using it.

### World/spatial

- small fixed test grid;
- cardinal outline collision;
- Chebyshev 8-direction movement;
- no diagonal corner cutting;
- spatial occupancy/proximity queries.

### Networking

- TCP;
- custom big-endian frame;
- typed codecs;
- bounded queues;
- movement/action/combat requests;
- position/animation/health/inventory updates as needed.

### Inventory

Start small:

- fixed-size slots;
- typed `ItemID`;
- `ItemStack`;
- add/get/set/swap/clear;
- one equippable weapon is enough.

Do not implement bank/trade/shop/container complexity yet.

### Actions

Actions should enter gameplay as typed requests and pass through simulation boundaries.

Do not let packet handlers become gameplay controllers.

### Animation

The server owns the authoritative animation event/state ID.

The client owns actual animation assets.

Server example:

```text
AnimationID 3
```

Client mapping example:

```text
3 -> attack_slash
```

### Combat

Start with one melee interaction:

- typed target entity;
- target validation;
- Chebyshev melee range;
- cooldown in ticks;
- fixed/simple damage;
- health reduction;
- attack animation;
- dead/unattackable state if needed.

A simple combat model that respects architecture is better than a sophisticated model that bypasses it.

## Explicitly out of scope for the first gameplay slice

Unless a ticket specifically requires one of these, do not pull it into unrelated MVP work:

- Google OAuth;
- Apple OAuth;
- production password auth;
- PostgreSQL;
- Redis;
- multiple real world processes;
- character `.sav` implementation;
- friends;
- clans;
- leaderboards;
- cross-world messaging;
- Lua scripting;
- quests;
- economy;
- trading;
- bank;
- production pathfinding;
- distributed simulation;
- Kubernetes/microservices.

Design package boundaries so these can be added later, but do not implement them speculatively.

## Concurrency cut line

First reach:

```text
correct
deterministic
tested
```

Then add:

- fixed worker pool;
- parallel Decide;
- worker-local intent buffers;
- phase barriers;
- parallel spatial rebuild if measured useful.

Do not require concurrency for the first proof that gameplay flows through the correct boundaries.

## Definition of a healthy vertical slice

A feature is not "done" merely because the client visually performs it.

For example, movement is healthy when:

```text
client input
  ↓
binary request
  ↓
typed decode
  ↓
bounded input path
  ↓
Decide
  ↓
MoveIntent
  ↓
Apply
  ↓
authoritative Position
  ↓
spatial rebuild
  ↓
outbound authoritative update
  ↓
client rendering
```

This pattern should be preserved for subsequent gameplay systems.
