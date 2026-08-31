# World Simulation

## Purpose

The game server runs an authoritative ECS-based simulation.

The world is logically 2D and uses a spatial grid/index for proximity queries.

The spatial grid is not the concurrency ownership model.

The simulation is designed around a phased tick:

```text
Decide
  ↓
Apply
  ↓
Rebuild Spatial Index
  ↓
Visibility / Replication
```

The target tick duration is currently:

```text
600 ms
```

All architecture should preserve the ability to change this duration later.

Avoid scattering hard-coded `600ms` values through gameplay systems.

---

# Core principle

Do not shard simulation ownership by world location.

Avoid designs such as:

```text
Chunk 1 goroutine owns all entities in Chunk 1
Chunk 2 goroutine owns all entities in Chunk 2
```

This produces hotspots when many entities gather in one location.

Example:

```text
Grand Exchange
500 entities
one chunk
one concurrency owner
```

Instead:

```text
entities are simulation work
location is indexed separately
```

The spatial index answers:

```text
what is near this position?
```

It does not answer:

```text
which goroutine owns this entity?
```

---

# Tick overview

Each tick proceeds through explicit phases.

```text
previous authoritative state
           │
           ▼
      build/read snapshot
           │
           ▼
        DECIDE
     parallel/read-only
           │
        barrier
           ▼
         APPLY
    authoritative mutation
           │
        barrier
           ▼
   REBUILD SPATIAL INDEX
           │
        barrier
           ▼
 VISIBILITY / REPLICATION
           │
           ▼
 next authoritative tick state
```

Each phase must fully complete before the next dependent phase begins.

---

# Phase 1: Decide

The Decide phase performs expensive gameplay reasoning.

Examples:

* player movement intent processing
* NPC AI
* pathfinding
* target selection
* combat action selection
* spell selection
* interaction decisions
* chase decisions
* wander decisions

Decide reads from an immutable/read-only snapshot of world state.

It must not mutate authoritative state.

---

# Parallelism

Decide should be parallelized across a fixed worker pool.

A reasonable initial worker count is around:

```text
runtime.GOMAXPROCS(0)
```

Exact tuning should be benchmark driven.

Conceptually:

```text
all entities/work
       │
       ▼
 shared work distribution
 ┌─────┼─────┬─────┐
 ▼     ▼     ▼     ▼
W1    W2    W3    W4
```

Entity location does not determine which worker handles it.

If 500 entities are on one tile, their Decide work should still be distributed across available workers.

---

# Snapshot

Decide operates on a read-only view of state from the previous completed authoritative phase.

Conceptually:

```text
Snapshot
├── positions
├── movement state
├── health
├── combat state
├── NPC state
├── spatial index
└── other read views
```

A snapshot does not necessarily mean deep-copying the entire ECS every tick.

Avoid:

```text
deep copy every component every 600ms
```

unless benchmarks demonstrate it is acceptable and simplicity justifies it.

Possible implementation strategies include:

* immutable read views
* double buffering
* generation/versioned storage
* selective copied data
* read-only ECS snapshots

Choose based on the ECS implementation.

The contract matters more than the mechanism:

```text
Decide cannot observe state mutating underneath it.
```

---

# Intents

Decide produces intents.

An intent represents:

```text
what an entity wants to do
```

rather than immediately changing the world.

Examples:

```text
MoveIntent
AttackIntent
InteractIntent
CastIntent
```

A movement intent might conceptually contain:

```go
type Intent struct {
    Entity EntityID
    From   Tile
    To     Tile
}
```

Feature packages should generally own their intent semantics.

For example:

```text
game/movement/
    intent.go
    decide.go
    apply.go

game/combat/
    intent.go
    decide.go
    apply.go
```

The simulation engine controls when these stages execute.

---

# Gameplay systems

Prefer ECS systems operating over matching component sets.

Examples:

```text
MovementDecisionSystem
    Position + Movement

NPCDecisionSystem
    NPC + Position + Behaviour

CombatDecisionSystem
    Combat + Target + Position
```

Avoid putting all gameplay behaviour into:

```text
Entity.Decide()
```

if doing so undermines ECS data-oriented structure.

The exact system API may evolve with the ECS implementation.

---

# Phase barrier

All Decide work must complete before Apply begins.

In Go this can use:

* `sync.WaitGroup`
* `errgroup`
* a fixed worker-pool barrier
* another explicit synchronization primitive

Do not let Apply begin while some workers are still reading the old snapshot and producing decisions.

---

# Phase 2: Apply

Apply mutates authoritative game state.

Examples:

* update position
* resolve movement conflict
* consume run energy
* apply damage
* start interaction
* change target
* modify object state
* spawn/despawn entities
* update inventory
* apply rewards

Apply is deliberately separated from expensive decision work.

---

# Local conflict resolution

Some mutations conflict spatially.

Example:

```text
Entity A wants tile 10,10
Entity B wants tile 10,10
```

Apply resolves this using authoritative game rules.

It may group intents by affected region/chunk/cell for efficient conflict processing.

This grouping is temporary.

Do not turn the bucket into permanent entity ownership.

Conceptually:

```text
MoveIntents
    ↓
partition by affected chunk
    ↓
resolve each bucket
    ↓
authoritative positions
```

---

# Apply parallelism

Apply does not initially need maximum parallelism.

The key observation is:

```text
Decide:
    expensive
    read-only
    easy to parallelize

Apply:
    usually cheap O(1)-style mutations
    conflicts possible
```

A hotspot may contain many cheap Apply operations, but this is generally preferable to introducing complex fine-grained locking around expensive AI/pathfinding.

Begin with simple deterministic application.

Parallelize Apply only when profiling demonstrates a need.

---

# Determinism

Where multiple intents conflict, resolution rules should be deterministic where practical.

Avoid relying on:

* goroutine scheduling
* map iteration order
* lock acquisition timing

to decide gameplay outcomes.

Potential deterministic strategies include ordering by:

* entity ID
* intent sequence
* explicit priority
* game-specific initiative

The exact policy belongs to the relevant gameplay feature.

---

# Movement

Movement uses tile-based logic.

The world supports eight-direction movement.

Directions:

```text
North
NorthEast
East
SouthEast
South
SouthWest
West
NorthWest
```

Chebyshev-style movement is appropriate for an 8-direction grid.

Movement validation remains server authoritative.

---

# Collision

The world uses outline/edge collision.

A tile can have directional blocking on its edges rather than only being globally blocked/unblocked.

Conceptually:

```text
     North
  ┌──────────┐
  │          │
W │   tile   │ E
  │          │
  └──────────┘
     South
```

Diagonal movement must consider the relevant edge/corner rules.

Collision logic should live in a dedicated gameplay/world capability rather than being duplicated across:

* player movement
* NPC movement
* pathfinding
* interaction range

---

# Running

Run state changes movement behaviour but remains part of authoritative simulation.

Client packet:

```text
SetRunEnabled
```

updates intended player mode.

The server determines whether running is actually permitted based on game state.

Future rules may include:

* run energy
* stun/root state
* movement restrictions
* terrain rules

Do not make the client authoritative because it has toggled run.

---

# Interaction

Interaction requests identify:

```text
runtime EntityID
action
```

For example:

```text
target tree EntityID 91822
action Chop
```

Apply/decision logic validates:

* entity exists
* entity is interactable
* target is in range
* player can reach target
* action is supported
* required equipment exists
* current action can be interrupted/replaced

Feature logic belongs under:

```text
game/interaction/
```

or the specific feature package.

---

# Phase 3: Spatial rebuild

After authoritative state mutations complete, rebuild the spatial index.

The spatial index exists for read-side queries.

Examples:

```text
entities at tile
entities near player
entities within radius
viewport entities
nearby NPCs
nearby objects
nearby ground items
```

---

# Spatial representation

Prefer cache-friendly structures.

Where world dimensions permit, prefer structures such as:

```text
slice of cells
cell → slice of EntityID
```

over unnecessarily allocation-heavy maps in hot per-tick paths.

For example:

```text
[]Cell
```

where:

```go
type Cell struct {
    Entities []EntityID
}
```

The exact implementation should be benchmarked against realistic entity counts.

---

# Rebuild strategy

Spatial rebuilding is naturally parallelizable.

Conceptually:

```text
authoritative positions
       │
       ▼
partition entities across workers
       │
 ┌─────┼─────┬─────┐
 ▼     ▼     ▼     ▼
local local local local
buffer buffer buffer buffer
       │
       ▼
      merge
       │
       ▼
new spatial index
```

Worker-local buffers avoid lock contention during insertion.

The implementation may use double buffering.

For example:

```text
current index
next index
```

Build into `next`, then swap:

```text
current ↔ next
```

After the swap, readers use the new immutable/current index.

---

# Allocation policy

Avoid unnecessary allocations inside the hard tick path.

Reuse buffers where practical.

Examples:

* intent slices
* worker queues
* spatial cell buffers
* temporary entity lists
* visibility buffers

Prefer:

```text
allocate once
reset length
reuse backing memory
```

instead of allocating entirely new structures every tick.

This reduces garbage collection pressure.

Do not introduce unsafe complexity purely to remove every allocation.

Profile first, but make hot-loop reuse the default design direction.

---

# Spatial chunk size

Chunk size exists for:

* map streaming
* query locality
* cache friendliness
* spatial grouping
* content organization

It is not a concurrency contract.

Therefore chunk sizing can later be tuned without rewriting the ownership model.

For example, 64x64 content regions may coexist with a smaller internal proximity cell size if that proves useful.

Do not assume:

```text
content region == simulation worker == spatial query cell
```

These are separate concepts.

---

# Visibility

After the spatial index reflects the new authoritative state, calculate what each player should know about.

Conceptually:

```text
player position
    ↓
spatial nearby query
    ↓
visibility rules
    ↓
current visible entity set
```

Compare this with the player's previously known set.

Produce:

```text
spawn
update
remove
```

events as required.

---

# Client viewport replication

Visibility determines content.

Networking transports it.

Keep those responsibilities separate.

Prefer:

```text
game/visibility
    decides which changes matter

engine/network
    encodes/sends them
```

rather than placing viewport gameplay rules inside network connection code.

---

# Parallel visibility

Visibility work is usually independent per player and may be parallelized.

For example:

```text
all players
    ↓
worker pool
    ↓
build replication updates
```

Do not let workers write directly to TCP sockets.

They should enqueue outbound messages through the connection abstraction.

---

# Tick timing

The target tick interval is 600ms.

Conceptually:

```text
tick start
    ↓
Decide
    ↓
Apply
    ↓
Spatial rebuild
    ↓
Visibility/replication
    ↓
sleep until next scheduled tick
```

Track phase durations independently.

Useful metrics:

```text
total tick duration
decide duration
apply duration
spatial rebuild duration
visibility duration
entity count
player count
NPC count
intent count
```

If a tick exceeds its deadline, report it.

Do not silently allow tick drift to accumulate indefinitely.

---

# Tick scheduling

Prefer scheduling against a target timeline rather than:

```text
run tick
sleep 600ms
```

because that produces:

```text
600ms + execution time
```

per tick.

Instead conceptually:

```text
nextTick += tickDuration
run tick
wait until nextTick
```

The implementation should define what happens if the server falls behind.

Possible future policies:

* log overrun
* immediately start next tick
* limited catch-up
* controlled degradation

Do not introduce complex catch-up behaviour until needed.

---

# Network input integration

Connection goroutines decode packets independently of the simulation.

They enqueue typed requests.

The simulation consumes those requests at a controlled point.

Conceptually:

```text
network goroutines
       │
       ▼
bounded gameplay input queue
       │
       ▼
tick input collection
       │
       ▼
Decide / Apply
```

Network goroutines must not directly mutate ECS state.

---

# Player input timing

Requests arriving during Tick N may be applied according to a clearly defined input cutoff.

A simple initial model is:

```text
collect inputs available at tick start
process them in this tick
later arrivals wait for next tick
```

The exact implementation can evolve.

The important property is predictable simulation sequencing.

---

# Server-authoritative movement and client presentation

The server remains authoritative despite the relatively slow tick.

The Godot client may use:

* interpolation
* prediction where appropriate
* destination markers
* animation immediately on input

for responsiveness.

But authoritative position comes from the server.

For remote entities, interpolation between authoritative updates is expected.

---

# Entity lifetime

Runtime ECS Entity IDs are temporary.

For example:

```text
CharacterID 1001
logs into World 2
    ↓
EntityID 91822
```

On logout:

```text
EntityID 91822 destroyed
```

On next login:

```text
CharacterID 1001
    ↓
EntityID 105991
```

Simulation code must not confuse persistent Character IDs with runtime Entity IDs.

---

# Spawning

Static world definitions such as:

* NPC spawns
* objects
* ground items

are read from content when the world loads.

The server converts them into authoritative runtime state.

Example:

```text
region definition
    ↓
groundItems itemId=1
    ↓
create runtime entity
    ↓
GroundItem component
Position component
DefinitionID component
```

The exact ECS component design should follow the chosen ECS implementation.

---

# Ground items

A region may define an initial ground item.

After world initialization, the ECS/runtime state is authoritative.

Example lifecycle:

```text
content spawn
    ↓
runtime ground item exists
    ↓
player picks it up
    ↓
Apply removes ground item entity/state
    ↓
spatial rebuild
    ↓
visibility sees removal
    ↓
client receives remove update
```

A future respawn system may recreate the item according to gameplay rules.

---

# Persistence boundary

Persistent character saving is not part of the hot entity decision loop.

Character state should be extracted at appropriate persistence points such as:

* logout
* periodic checkpoint
* important transaction
* server shutdown

Avoid synchronous heavy file I/O directly inside latency-sensitive tick mutation code.

Persistence design must still preserve correctness.

---

# Package responsibilities

A likely package relationship is:

```text
engine/ecs
    entity/component storage

engine/simulation
    tick sequencing
    workers
    barriers
    snapshots
    reusable tick buffers

engine/spatial
    proximity/index queries

game/movement
    movement rules and intents

game/combat
    combat rules and intents

game/npc
    NPC decision logic

game/interaction
    interaction rules

game/visibility
    replication interest decisions

engine/network
    packet transport
```

Core rule:

```text
simulation controls WHEN

game controls WHAT

spatial answers WHERE

ECS stores CURRENT STATE

network transports INFORMATION
```

---

# Initial implementation priority

Do not build the entire concurrency system before basic gameplay works.

A reasonable incremental order is:

```text
1. deterministic single-thread tick
2. movement intents
3. Apply movement
4. spatial rebuild
5. visibility
6. basic NPC decisions
7. worker pool for Decide
8. buffer reuse
9. profile
10. parallelize additional phases only when useful
```

The architecture should permit parallel Decide from the beginning, but simplicity during early MVP development is valuable.

---

# Testing

Simulation tests should cover:

* deterministic movement
* collision
* diagonal movement
* conflicting movement destinations
* invalid moves
* interaction range
* missing targets
* spatial index rebuild correctness
* visibility spawn/remove transitions
* entity destruction
* concurrent Decide producing deterministic Apply inputs
* tick phase ordering

Avoid tests that rely on goroutine timing.

Prefer explicit barriers and deterministic inputs.

---

# Performance principle

Optimize the expensive part first.

The intended performance model is:

```text
Decide
    expensive
    highly parallel

Apply
    cheap
    deterministic/local

Spatial rebuild
    parallelizable

Visibility
    parallelizable by player
```

A hotspot containing many entities should produce:

```text
more work
```

rather than:

```text
more lock contention
```

That is the central concurrency goal of this simulation design.
