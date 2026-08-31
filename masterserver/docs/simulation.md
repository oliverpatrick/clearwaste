# Simulation and Tick Model

## Tick duration

The current target tick duration is 600 ms.

Treat the tick interval as configuration, but gameplay timing should be expressed in ticks where deterministic simulation behaviour is required.

## Phases

A tick is split into explicit phases.

```text
authoritative state from tick N
            │
            ▼
       build read view
            │
            ▼
┌─────────────────────────┐
│        DECIDE           │
│ read-only snapshot      │
│ expensive work          │
│ parallel by work/entity │
└────────────┬────────────┘
             │ intents
             ▼
       phase barrier
             │
             ▼
┌─────────────────────────┐
│         APPLY           │
│ mutate authoritative    │
│ resolve conflicts       │
│ deterministic ordering  │
└────────────┬────────────┘
             │
             ▼
       phase barrier
             │
             ▼
┌─────────────────────────┐
│        REBUILD          │
│ spatial read-side index │
└────────────┬────────────┘
             │
             ▼
      visibility/outbound
```

## Decide

The Decide phase should contain the expensive work that benefits most from parallelism.

Examples:

- path selection;
- NPC target selection;
- movement validation against the snapshot;
- combat target/range/cooldown decisions;
- interaction selection.

Rules:

- only read immutable tick snapshot data;
- do not mutate ECS components;
- do not mutate the shared spatial index;
- produce typed intents into worker-local or safely partitioned buffers;
- avoid blocking network/database operations.

A pile-up of entities in one spatial area should not create lock contention during Decide.

## Apply

Apply is the authoritative mutation phase.

Examples:

- update position;
- resolve two entities trying to occupy an exclusive tile;
- apply damage;
- change inventory/equipment;
- set animation state;
- create/remove runtime entities.

Rules:

- deterministic conflict resolution;
- no dependency on goroutine scheduling order;
- prefer cheap O(1) mutations;
- partition conflict domains if/when measurements justify it.

If intents are temporarily bucketed by chunk/cell for conflict resolution, that is an Apply implementation detail. It does not make the chunk the permanent owner of entities.

## Rebuild

After Apply, rebuild or incrementally refresh the spatial read-side index from authoritative positions.

The preferred initial design is simple and predictable:

```text
positions
  ↓
clear/reuse spatial buckets
  ↓
insert current entities
  ↓
publish updated index for queries
```

If rebuild cost later matters, it may be parallelised using worker-local buffers and a merge.

## Snapshot design

"Snapshot" means immutable simulation input, not necessarily a deep copy of every ECS component.

Avoid a naïve full-world deep copy every 600 ms unless measurement shows it is acceptable and simplicity is worth it.

Possible implementations include:

- read-only views over state frozen for the phase;
- double-buffered component storage;
- compact copied arrays for only components used during Decide;
- generation/versioned views.

The API must make mutation impossible or clearly invalid during Decide.

## Worker pool

Use a fixed worker pool sized around available CPU, not one goroutine per entity.

Typical target:

```go
workers ~= runtime.GOMAXPROCS(0)
```

Phase barriers are mandatory.

Decide must finish before Apply begins.

Apply must finish before post-Apply spatial state is published.

`sync.WaitGroup`, `errgroup`, or a purpose-built worker pool are acceptable if lifecycle and error behaviour remain clear.

## Per-tick allocation

The tick path should avoid unnecessary allocation churn.

Prefer:

- preallocated intent slices;
- worker-local reusable buffers;
- spatial cell slices whose capacity is reused;
- stable ECS storage;
- explicit reset methods.

Avoid:

- rebuilding many maps every tick;
- `map[string]any`;
- per-entity goroutine creation;
- converting stable numeric IDs to strings in hot loops.

## Spatial index

The spatial index answers proximity questions.

A grid implementation can use a flat array/slice of cells for a known world layout.

Example concept:

```text
cell index = y*width + x
cells[index] -> []EntityID
```

Choose representation based on query locality and measured cost.

Do not choose chunk size based on concurrency ownership; choose it based on indexing/query needs.

## Movement

Movement uses Chebyshev adjacency.

For one step:

```text
max(abs(dx), abs(dy)) == 1
```

Cardinal collision is stored on edges:

```text
North
East
South
West
```

Diagonal movement must validate both contributing cardinal edges.

Examples:

- NE requires North and East open;
- SE requires South and East open;
- SW requires South and West open;
- NW requires North and West open.

## Determinism

Where multiple intents conflict, ordering must be explicit.

Do not rely on:

- Go map iteration order;
- goroutine completion order;
- channel race order.

If an order is needed, define one, such as:

- entity ID;
- intent sequence captured at input time;
- stable source priority;
- explicit tie-break rule.

Document gameplay-significant tie-break rules near the owning feature.

## Network interaction with ticks

Network goroutines accept and decode input continuously, but authoritative gameplay input is consumed at a simulation boundary.

Conceptually:

```text
reader goroutine
   ↓
typed request
   ↓
bounded inbound queue
   ↓
tick input capture
   ↓
Decide
```

A network packet should never mutate position/health/inventory directly.

## Visibility

After Apply and spatial update, visibility can query the latest index and produce outbound changes.

Examples:

- entity entered viewport -> spawn;
- entity stayed visible and moved -> movement update;
- entity left viewport -> remove.

Keep visibility decisions outside low-level transport code.
