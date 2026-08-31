# Gameplay Request Protocol Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decode authenticated Move, SetRunEnabled, and Interact requests into the existing bounded session inbound queue without mutating gameplay state.

**Architecture:** Explicit stable application opcodes live in a neutral network-contract package. Movement and interaction packages own concrete request types and codecs; those types implement the existing marker-only `network.GameplayMessage`, so the current login handler gates them by session state before the existing per-session queue.

**Tech Stack:** Go 1.25.1, standard library, existing custom big-endian binary protocol and typed codec registry.

**Spec:** `docs/superpowers/specs/2026-08-31-gameplay-request-protocol-design.md`

## Global Constraints

- Keep existing login wire values 1 through 5 unchanged.
- Use `MoveRequest=6`, `SetRunEnabled=7`, and `InteractRequest=8`.
- Fixed payload sizes are exactly 1, 1, and 9 bytes respectively.
- Networking validates representation only and performs no gameplay/ECS mutation.
- Use the existing session inbound capacity and overflow-disconnect policy; add no queue or configuration.
- Send no gameplay acknowledgements.
- Add no simulation, collision, stamina, harvesting, inventory, animation, or reward logic.
- Preserve all pre-existing untracked files.

---

### Task 1: Stable Opcode Contract

**Files:**
- Create: `internal/engine/network/opcode/opcode.go`
- Create: `internal/engine/network/opcode/opcode_test.go`
- Modify: `internal/world/login/packets.go`

**Interfaces:**
- Produces: `opcode.ClientHello`, `opcode.ServerHello`, `opcode.LoginRequest`, `opcode.LoginAccepted`, `opcode.LoginRejected`, `opcode.MoveRequest`, `opcode.SetRunEnabled`, and `opcode.InteractRequest`, all typed as `protocol.Opcode`.
- Preserves: existing `login.Opcode*` exported constants as aliases so current callers and tests remain valid.

- [ ] **Step 1: Write failing exact-value and uniqueness tests**

Create `internal/engine/network/opcode/opcode_test.go`:

```go
package opcode

import (
	"testing"

	"master/clearwaste/internal/engine/network/protocol"
)

func TestValuesAreStableAndUnique(t *testing.T) {
	tests := []struct {
		name string
		got  protocol.Opcode
		want protocol.Opcode
	}{
		{"ClientHello", ClientHello, 1},
		{"ServerHello", ServerHello, 2},
		{"LoginRequest", LoginRequest, 3},
		{"LoginAccepted", LoginAccepted, 4},
		{"LoginRejected", LoginRejected, 5},
		{"MoveRequest", MoveRequest, 6},
		{"SetRunEnabled", SetRunEnabled, 7},
		{"InteractRequest", InteractRequest, 8},
	}
	seen := make(map[protocol.Opcode]string, len(tests))
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s=%d want=%d", test.name, test.got, test.want)
		}
		if previous, exists := seen[test.got]; exists {
			t.Errorf("%s and %s share opcode %d", previous, test.name, test.got)
		}
		seen[test.got] = test.name
	}
}
```

- [ ] **Step 2: Run the test to verify RED**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/opcode`

Expected: compile failure because the opcode constants do not exist.

- [ ] **Step 3: Add explicit constants and preserve login aliases**

Create `internal/engine/network/opcode/opcode.go`:

```go
// Package opcode defines the stable application-level wire opcode contract.
package opcode

import "master/clearwaste/internal/engine/network/protocol"

const (
	ClientHello   protocol.Opcode = 1
	ServerHello   protocol.Opcode = 2
	LoginRequest  protocol.Opcode = 3
	LoginAccepted protocol.Opcode = 4
	LoginRejected protocol.Opcode = 5

	MoveRequest   protocol.Opcode = 6
	SetRunEnabled protocol.Opcode = 7
	InteractRequest protocol.Opcode = 8
)
```

Replace the `iota` constants in `internal/world/login/packets.go` with aliases to the neutral package while leaving packet types and methods unchanged:

```go
const (
	OpcodeClientHello   = opcode.ClientHello
	OpcodeServerHello   = opcode.ServerHello
	OpcodeLoginRequest  = opcode.LoginRequest
	OpcodeLoginAccepted = opcode.LoginAccepted
	OpcodeLoginRejected = opcode.LoginRejected
)
```

- [ ] **Step 4: Format and verify GREEN plus unchanged login behaviour**

Run: `gofmt -w internal/engine/network/opcode internal/world/login/packets.go`

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/opcode ./internal/world/login`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/network/opcode internal/world/login/packets.go
git commit -m "feat: centralize stable application opcodes"
```

### Task 2: Movement and Run Requests

**Files:**
- Create: `internal/game/movement/requests.go`
- Create: `internal/game/movement/packets.go`
- Create: `internal/game/movement/packets_test.go`

**Interfaces:**
- Consumes: `opcode.MoveRequest`, `opcode.SetRunEnabled`, `network.Register`, `protocol.Reader`, and `protocol.Writer`.
- Produces: `movement.Direction`, `movement.MoveRequest`, `movement.SetRunEnabled`, and `movement.RegisterCodecs(*network.Registry) error`.

- [ ] **Step 1: Write failing movement/run codec tests**

Create tests that use a real `network.Registry` and assert:

```go
func TestMoveRequestRoundTripsAllDirections(t *testing.T) {
	directions := []Direction{North, NorthEast, East, SouthEast, South, SouthWest, West, NorthWest}
	for value, direction := range directions {
		if uint8(direction) != uint8(value) {
			t.Fatalf("direction %v=%d want=%d", direction, direction, value)
		}
		assertRoundTrip(t, MoveRequest{Direction: direction})
	}
}

func TestMoveRequestRejectsInvalidDirection(t *testing.T) {
	for _, value := range []byte{8, 255} {
		_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest, Payload: []byte{value}})
		if !errors.Is(err, ErrInvalidDirection) {
			t.Fatalf("direction=%d error=%v", value, err)
		}
	}
}

func TestMoveRequestRequiresExactlyOneByte(t *testing.T) {
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest})
	if !errors.Is(truncated, protocol.ErrUnderflow) { t.Fatalf("truncated error=%v", truncated) }
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.MoveRequest, Payload: []byte{0, 0}})
	if !errors.Is(trailing, network.ErrTrailingPayload) { t.Fatalf("trailing error=%v", trailing) }
}

func TestSetRunEnabledDecodesZeroAndOne(t *testing.T) {
	for _, test := range []struct{ wire byte; want bool }{{0, false}, {1, true}} {
		message, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{test.wire}})
		if err != nil { t.Fatal(err) }
		if got := message.(SetRunEnabled).Enabled; got != test.want { t.Fatalf("enabled=%t want=%t", got, test.want) }
	}
}

func TestSetRunEnabledRejectsInvalidOrWrongLengthPayload(t *testing.T) {
	_, invalid := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{2}})
	if !errors.Is(invalid, protocol.ErrInvalidBool) { t.Fatalf("invalid error=%v", invalid) }
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled})
	if !errors.Is(truncated, protocol.ErrUnderflow) { t.Fatalf("truncated error=%v", truncated) }
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.SetRunEnabled, Payload: []byte{1, 0}})
	if !errors.Is(trailing, network.ErrTrailingPayload) { t.Fatalf("trailing error=%v", trailing) }
}
```

Include helpers that register both movement codecs and compare an encoded/decoded message with `reflect.DeepEqual`.

- [ ] **Step 2: Run the tests to verify RED**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/game/movement`

Expected: compile failure because the movement package types do not exist.

- [ ] **Step 3: Add the minimal request types and codecs**

Define the request contract in `requests.go`:

```go
package movement

type Direction uint8

const (
	North Direction = iota
	NorthEast
	East
	SouthEast
	South
	SouthWest
	West
	NorthWest
)

type MoveRequest struct { Direction Direction }
type SetRunEnabled struct { Enabled bool }
```

In `packets.go`, add `ErrInvalidDirection`, opcode methods, empty `Gameplay()` marker methods, `RegisterCodecs`, and codecs. Decode one byte and reject values above `NorthWest`; use `Reader.Bool` for run mode. Validate direction on encode as well. Do not reference any position, running component, collision, or tick type.

- [ ] **Step 4: Format and verify GREEN**

Run: `gofmt -w internal/game/movement`

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/game/movement ./internal/world/login`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/game/movement
git commit -m "feat: add movement gameplay requests"
```

### Task 3: Runtime Entity ID and Interaction Request

**Files:**
- Create: `internal/game/entity/id.go`
- Create: `internal/game/entity/id_test.go`
- Create: `internal/game/interaction/requests.go`
- Create: `internal/game/interaction/packets.go`
- Create: `internal/game/interaction/packets_test.go`

**Interfaces:**
- Consumes: `opcode.InteractRequest`, `network.Register`, `protocol.Reader`, and `protocol.Writer`.
- Produces: canonical network-visible `entity.ID uint64`, reserved `entity.Invalid`, `interaction.Action`, `interaction.InteractRequest`, and `interaction.RegisterCodecs(*network.Registry) error`.

- [ ] **Step 1: Write failing entity and interaction tests**

Test the ID invariant:

```go
func TestZeroIsReservedInvalid(t *testing.T) {
	if Invalid != ID(0) { t.Fatalf("Invalid=%d", Invalid) }
	if Invalid.Valid() { t.Fatal("zero entity ID is valid") }
	if !ID(1).Valid() { t.Fatal("non-zero entity ID is invalid") }
}
```

Test interaction codecs through a real registry:

```go
func TestInteractRequestRoundTripsChopAndMine(t *testing.T) {
	for _, action := range []Action{Chop, Mine} {
		assertRoundTrip(t, InteractRequest{TargetID: entity.ID(4812), Action: action})
	}
}

func TestInteractRequestRejectsInvalidActions(t *testing.T) {
	for _, action := range []byte{0, 3, 255} {
		payload := []byte{0, 0, 0, 0, 0, 0, 0, 1, action}
		_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: payload})
		if !errors.Is(err, ErrInvalidAction) { t.Fatalf("action=%d error=%v", action, err) }
	}
}

func TestInteractRequestRejectsZeroTarget(t *testing.T) {
	_, err := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: []byte{0, 0, 0, 0, 0, 0, 0, 0, byte(Chop)}})
	if !errors.Is(err, ErrInvalidTargetID) { t.Fatalf("error=%v", err) }
}

func TestInteractRequestRequiresExactlyNineBytes(t *testing.T) {
	_, truncated := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: []byte{0, 0, 0, 0, 0, 0, 0, 1}})
	if !errors.Is(truncated, protocol.ErrUnderflow) { t.Fatalf("truncated error=%v", truncated) }
	_, trailing := newRegistry(t).Decode(protocol.Frame{Opcode: opcode.InteractRequest, Payload: []byte{0, 0, 0, 0, 0, 0, 0, 1, byte(Mine), 0}})
	if !errors.Is(trailing, network.ErrTrailingPayload) { t.Fatalf("trailing error=%v", trailing) }
}
```

- [ ] **Step 2: Run the tests to verify RED**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/game/entity ./internal/game/interaction`

Expected: compile failure because entity and interaction types do not exist.

- [ ] **Step 3: Add the minimal entity and interaction contracts**

Create `entity/id.go`:

```go
// Package entity defines runtime entity identity crossing gameplay boundaries.
package entity

// ID identifies one spawned runtime world entity, independently of ECS handles.
type ID uint64

// Invalid is reserved and never identifies a runtime entity.
const Invalid ID = 0

// Valid reports whether the ID may identify a runtime entity.
func (id ID) Valid() bool { return id != Invalid }
```

Define `Action` with `Chop=1`, `Mine=2`, and `InteractRequest { TargetID entity.ID; Action Action }`. In `packets.go`, read `Uint64` then `Uint8` before semantic validation, reject zero target and actions outside one through two, add opcode and marker methods, register the codec, and perform the same validation on encode. Add no target lookup or result fields.

- [ ] **Step 4: Format and verify GREEN**

Run: `gofmt -w internal/game/entity internal/game/interaction`

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/game/entity ./internal/game/interaction`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/game/entity internal/game/interaction
git commit -m "feat: add environmental interaction requests"
```

### Task 4: Server Registration and Authenticated Queue Integration

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/world/login/handler_test.go`
- Create: `internal/world/login/gameplay_test.go`
- Create: `docs/architecture/gameplay-request-protocol.md`

**Interfaces:**
- Consumes: `movement.RegisterCodecs`, `interaction.RegisterCodecs`, existing login handler/session state, and existing `WORLD_MAX_PAYLOAD_BYTES`, `WORLD_INBOUND_QUEUE_CAPACITY`, and `WORLD_PROTOCOL_VERSION` configuration flow.
- Produces: a production registry containing login and all three gameplay requests, plus authenticated end-to-end queue coverage.

- [ ] **Step 1: Write failing authenticated-boundary tests**

Extend the test registry setup in `handler_test.go` to call both new feature registration functions. Add `gameplay_test.go` tests that use the existing `net.Pipe` connection helpers:

```go
func TestConcreteGameplayRequestsRequireGameState(t *testing.T) {
	requests := []network.Message{
		movement.MoveRequest{Direction: movement.North},
		movement.SetRunEnabled{Enabled: true},
		interaction.InteractRequest{TargetID: entity.ID(4812), Action: interaction.Chop},
	}
	for _, request := range requests {
		connection, peer, registry := startLoginConnection(t, configuredValidator())
		sendLoginMessage(t, peer, registry, request)
		if err := connection.Wait(); !errors.Is(err, ErrGameplayBeforeLogin) {
			t.Fatalf("%T error=%v", request, err)
		}
	}
}

func TestConcreteGameplayRequestsEnterAuthenticatedInboundInOrder(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeLogin(t, connection, peer, registry)
	want := []network.Message{
		movement.MoveRequest{Direction: movement.NorthEast},
		movement.SetRunEnabled{Enabled: true},
		interaction.InteractRequest{TargetID: entity.ID(4812), Action: interaction.Mine},
	}
	identityBefore, _ := connection.Session().Identity()
	for _, request := range want { sendLoginMessage(t, peer, registry, request) }
	for _, expected := range want {
		select {
		case got := <-connection.Session().Inbound():
			if !reflect.DeepEqual(got, expected) { t.Fatalf("got=%#v want=%#v", got, expected) }
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %T", expected)
		}
	}
	identityAfter, _ := connection.Session().Identity()
	if connection.Session().State() != network.StateGame || identityAfter != identityBefore {
		t.Fatalf("network request mutated session game identity/state")
	}
}
```

Add a capacity-aware test helper, authenticate a noisy and healthy connection with capacity one, fill only the noisy queue, assert its `Wait()` returns `network.ErrInboundBackpressure`, then send and receive one request on the healthy connection and assert its `Done()` remains open.

- [ ] **Step 2: Run integration tests to verify RED**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login -run 'ConcreteGameplay|GameplayQueueOverflow'`

Expected: failure because the login test registry and production server do not yet register feature codecs or the capacity-aware helper is absent.

- [ ] **Step 3: Register feature codecs in tests and production**

In `cmd/server/main.go`, after `login.RegisterCodecs`, register `movement.RegisterCodecs` and `interaction.RegisterCodecs`, returning startup errors through the existing `log.Fatal` path. Do not add configuration or acknowledgements.

Update login test registry construction to register the same feature codecs. Keep existing private marker tests if they still add coverage of unclassified-message rejection.

- [ ] **Step 4: Document the exact contract**

Create `docs/architecture/gameplay-request-protocol.md` containing:

- opcode table values 1 through 8;
- Move payload `[direction uint8]` and all values 0 through 7;
- Run payload `[enabled uint8]`, only 0 and 1;
- Interact payload `[target entity ID uint64 big-endian][action uint8]`, Chop 1 and Mine 2;
- entity zero reservation and identity distinctions;
- authenticated queue flow and overflow-disconnect policy;
- explicit statement that queueing does not imply success or mutate gameplay;
- future `Session.Inbound -> tick drain -> Snapshot -> Decide -> Apply` handoff.

- [ ] **Step 5: Format and run focused GREEN checks**

Run: `gofmt -w cmd/server/main.go internal/world/login/handler_test.go internal/world/login/gameplay_test.go`

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./cmd/server ./internal/world/login ./internal/game/... ./internal/engine/network/opcode`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go internal/world/login/handler_test.go internal/world/login/gameplay_test.go docs/architecture/gameplay-request-protocol.md
git commit -m "feat: route gameplay requests to authenticated sessions"
```

### Task 5: Repository Verification and Scope Audit

**Files:**
- Review only; modify only files from Tasks 1 through 4 if a verification failure exposes a defect.

**Interfaces:**
- Verifies the complete gameplay request contract and all existing transport/login behaviour.

- [ ] **Step 1: Run repository tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -count=1 ./...`

Expected: PASS.

- [ ] **Step 2: Run static analysis**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go vet ./...`

Expected: exit 0.

- [ ] **Step 3: Run race detection**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race -count=1 ./...`

Expected: PASS with no race reports.

- [ ] **Step 4: Audit scope and repository cleanliness**

Run: `git diff --check`

Run: `git status --short`

Confirm no new configuration, acknowledgements, ECS state, collision, target lookup, simulation processing, or unrelated file changes were introduced. Confirm the pre-existing untracked files remain untouched.
