# Testing and Development Conventions

## Goal

Tests should protect architecture boundaries as well as gameplay results.

A passing test suite should make it difficult to accidentally:

- mutate state in the wrong phase;
- bypass collision;
- bypass protocol validation;
- couple persistence to ECS internals;
- introduce scheduling-dependent behaviour.

## Test layers

### Pure rule tests

Best for:

- Chebyshev distance;
- direction conversion;
- edge collision;
- cooldown calculations;
- inventory slot rules;
- item definition validation;
- save migrations.

These should be fast and require minimal setup.

### Package/system tests

Best for:

- movement Decide producing an intent;
- movement Apply changing a position;
- combat validation;
- equipment changing combat properties;
- visibility state transitions.

### Tick integration tests

Build a minimal world and run complete ticks.

Examples:

```text
request move
  ↓
tick
  ↓
position changed
  ↓
spatial cell changed
```

or:

```text
attack request
  ↓
Decide creates attack intent
  ↓
Apply damages target
  ↓
animation event generated
```

### Protocol tests

Every codec should have round-trip and malformed-input coverage.

### Persistence tests

Use fixtures for each supported save version.

Test:

- encode/decode;
- migration;
- corrupt/truncated save rejection;
- atomic generation/pointer behaviour at repository level.

## Simulation invariants to test

### Decide cannot mutate authoritative state

Given a pre-tick state, running Decide should leave the authoritative ECS unchanged.

### Apply is deterministic

Run equivalent initial state + equivalent ordered input more than once and assert identical output.

### Spatial data is post-Apply data

After a move is applied and Rebuild completes:

- old cell no longer contains the entity;
- new cell contains the entity.

### Collision is symmetric

For adjacent tiles A and B:

```text
A East blocked
```

must prevent traversal from A to B and the equivalent traversal from B to A via West.

### No diagonal corner cutting

If either relevant cardinal edge is blocked, the diagonal move fails.

## Concurrency testing

For changes involving goroutines, queues, worker pools, or phase barriers:

```bash
go test -race ./...
```

Do not "fix" a race by adding a broad global mutex unless the ownership model actually requires it.

Prefer:

- immutable snapshot reads;
- worker-local buffers;
- phase barriers;
- clearly owned mutable structures.

## Protocol fuzz/property testing

Good candidates for fuzzing later:

- frame decoder;
- packet codecs;
- save decoder;
- content binary decoder.

The decoder should never panic on arbitrary bytes.

## Benchmarks

Do not benchmark everything.

Useful hot-path benchmarks include:

- ECS queries;
- spatial rebuild;
- `Nearby` query;
- frame encode/decode;
- movement/combat Decide over many entities;
- full tick with representative entity counts.

Measure allocations:

```bash
go test -bench . -benchmem
```

Optimisation should follow measurement.

## Commands

Minimum local verification:

```bash
go test ./...
go vet ./...
```

Concurrency changes:

```bash
go test -race ./...
```

Benchmarks where relevant:

```bash
go test -bench . -benchmem ./...
```

## Test naming

Prefer behaviour-oriented names.

Good:

```text
TestCanTraverseNorth_WhenEdgeOpen
TestCannotTraverseNorth_WhenNorthEdgeBlocked
TestMoveDecide_DoesNotMutatePosition
TestMoveApply_UpdatesPosition
TestFrameDecoder_RejectsOversizedPayload
```

Avoid meaningless names such as:

```text
TestMovement1
TestThing
TestHandler
```

## Fixtures

Keep fixtures small and purpose-built.

Avoid using a full production-size map/content pack for a movement unit test.

A 3x3 or 5x5 grid is usually enough to prove cardinal/diagonal collision.

## Agent change checklist

Before marking a task complete:

- [ ] behaviour has an owning package;
- [ ] relevant tests added/updated;
- [ ] focused tests pass;
- [ ] `go test ./...` passes;
- [ ] `go vet ./...` passes;
- [ ] race detector run if concurrency changed;
- [ ] wire format docs updated if protocol changed;
- [ ] persistence docs/migrations updated if save schema changed;
- [ ] architecture docs updated if ownership/boundary changed.
