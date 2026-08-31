# AI Agent Task Checklist

Use this when implementing or reviewing a change.

## 1. Locate ownership

Which package owns this behaviour?

Examples:

```text
movement rule       -> game/movement
collision query     -> engine/spatial
tick sequencing     -> engine/simulation
socket framing      -> engine/network
character save      -> character/save
world capacity      -> world
authentication      -> account/auth
```

If ownership is unclear, fix the boundary before adding a cross-package shortcut.

## 2. Identify contracts affected

Does this task change any of these?

- wire protocol;
- opcode values;
- save schema;
- content schema;
- ECS component semantics;
- tick phase behaviour;
- public repository/service interface.

If yes, tests and docs should change with the code.

## 3. Preserve authoritative flow

For gameplay input, verify:

```text
network input
  ↓
typed request
  ↓
simulation
  ↓
authoritative state mutation
```

Reject implementations that skip the simulation boundary.

## 4. Check identity type

Ask which identity is correct:

- AccountID?
- CharacterID?
- EntityID?
- WorldID?
- content definition ID?

Do not substitute a name/string because it is convenient.

## 5. Check tick phase

For each state access ask:

- read in Decide?
- mutate in Apply?
- derive/rebuild after Apply?

If a Decide function needs a mutex to mutate shared game state, the design is probably wrong.

## 6. Check allocation/concurrency

For hot tick code:

- avoid per-entity goroutines;
- reuse buffers when practical;
- avoid maps when a stable indexed slice is simpler;
- do not add locks to compensate for unclear ownership.

## 7. Test the smallest rule first

Add narrow tests before or alongside broad integration tests.

Examples:

```text
collision edge
Chebyshev range
codec enum validation
inventory slot rule
save migration
```

Then add the vertical integration test.

## 8. Verify

Run:

```bash
go test ./...
go vet ./...
```

If concurrency changed:

```bash
go test -race ./...
```

## 9. Update documentation

Update relevant docs when changing:

- package ownership;
- process boundaries;
- protocol layout/opcodes;
- save format/version;
- simulation phase rules;
- core gameplay invariants.

## 10. Avoid opportunistic scope expansion

Do not introduce:

- microservices;
- Redis;
- PostgreSQL;
- OAuth;
- scripting;
- distributed ECS;
- new generic framework layers;

unless the current task requires them.

Keep the change small, coherent, and reversible.
