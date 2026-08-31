# World Login and Handshake Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish a versioned world handshake and development-only opaque-ticket login that authenticates a session before typed gameplay messages can reach its simulation-facing queue.

**Architecture:** The existing connection decodes packets and delegates them to a state-aware `InboundHandler`. `internal/world/login` consumes handshake/login packets, validates an opaque ticket through a replaceable validator, updates authoritative session state, and allows only explicitly marked gameplay messages through.

**Tech Stack:** Go 1.25 standard library only (`crypto/subtle`, `encoding/binary`, `errors`, `sync`, `testing`).

**Spec:** `docs/superpowers/specs/2026-08-31-world-login-handshake-design.md`

## Global Constraints

- Generic framing and binary primitives remain in `internal/engine/network/protocol`; login packets do not.
- Initial session state is `Handshake`; legal progress is `Handshake -> Login -> Game -> Closed`.
- `Session` is the sole authority for state transitions and authenticated identity mutation.
- The client sends only an opaque ticket, never `AccountID` or `CharacterID`.
- Login and handshake messages are consumed and never reach `Session.Inbound()`.
- Only `network.GameplayMessage` values in `Game` may reach `Session.Inbound()`.
- All outbound responses use the existing bounded outbound queue and sole writer goroutine.
- Development ticket values are never logged or stored in sessions.
- `WORLD_PROTOCOL_VERSION` defaults to `1`; development ticket and identity have no source defaults.
- Missing or partial development configuration rejects every login; any supplied malformed or zero ID fails configuration loading.
- Add no OAuth, database, world/build identity, ECS spawning, gameplay, or external dependency.

## File Map

- `internal/account/id.go`: strongly typed account identifier.
- `internal/character/id.go`: strongly typed character identifier.
- `config/config.go`: protocol version and optional complete development-login mapping.
- `config/config_test.go`: defaults, complete mapping, partial/unconfigured fallback, and invalid IDs.
- `internal/engine/network/inbound.go`: inbound handler and gameplay marker contracts.
- `internal/engine/network/session.go`: protocol state, typed identity, outbound delegate, and atomic authentication.
- `internal/engine/network/connection.go`: handler invocation, state-safe enqueue, and session closure.
- `internal/engine/network/server.go`: handler propagation to accepted connections.
- `internal/world/login/packets.go`: login opcodes and typed packet definitions.
- `internal/world/login/codecs.go`: packet codec registration.
- `internal/world/login/validator.go`: validator interface and development-only implementation.
- `internal/world/login/handler.go`: state-aware handshake, login, and gameplay routing.
- `cmd/server/main.go`: config, codecs, validator, handler, and server composition.
- `docs/architecture/world-login-handshake.md`: permanent architecture and replacement-seam documentation.

---

### Task 1: Typed Identity and Environment-Backed Login Configuration

**Files:**
- Create: `internal/account/id.go`
- Create: `internal/character/id.go`
- Modify: `config/config.go`
- Modify: `config/config_test.go`

**Interfaces:**
- Produces: `account.ID uint64` and `character.ID uint64`.
- Produces: `Config.ProtocolVersion uint16`.
- Produces: `Config.DevelopmentLogin *DevelopmentLoginConfig` where the nested config has `Ticket string`, `AccountID uint64`, and `CharacterID uint64`.
- Consumes environment variables: `WORLD_PROTOCOL_VERSION`, `WORLD_DEV_LOGIN_TICKET`, `WORLD_DEV_ACCOUNT_ID`, and `WORLD_DEV_CHARACTER_ID`.

- [ ] **Step 1: Write failing configuration tests**

Add tests proving protocol version defaults to literal `1`, supports an environment override, rejects zero/non-numeric versions, builds a complete development mapping, leaves it nil when all values are absent, leaves it nil for every partial combination, and rejects each supplied zero/non-numeric ID even when the other values are absent.

```go
func TestCompleteDevelopmentLoginConfiguration(t *testing.T) {
	t.Setenv("WORLD_DEV_LOGIN_TICKET", "opaque")
	t.Setenv("WORLD_DEV_ACCOUNT_ID", "41")
	t.Setenv("WORLD_DEV_CHARACTER_ID", "73")
	cfg, err := Load()
	if err != nil { t.Fatal(err) }
	if cfg.DevelopmentLogin == nil || cfg.DevelopmentLogin.Ticket != "opaque" || cfg.DevelopmentLogin.AccountID != 41 || cfg.DevelopmentLogin.CharacterID != 73 {
		t.Fatalf("development login=%+v", cfg.DevelopmentLogin)
	}
}
```

- [ ] **Step 2: Run the config tests and witness the missing API failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./config -run 'Test(Protocol|CompleteDevelopment|PartialDevelopment|UnconfiguredDevelopment|InvalidDevelopment)'`

Expected: build failure because `ProtocolVersion` and `DevelopmentLogin` do not exist.

- [ ] **Step 3: Implement the minimal config loader**

Add `envUint16` and an optional development-login loader. Parse every non-empty ID with `strconv.ParseUint(..., 10, 64)` and reject zero. Return nil unless ticket and both valid IDs are present. Do not change `.env.example`.

- [ ] **Step 4: Run all configuration tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./config`

Expected: PASS.

- [ ] **Step 5: Add the two leaf ID types**

```go
package account
type ID uint64
```

```go
package character
type ID uint64
```

- [ ] **Step 6: Commit**

```bash
git add config/config.go config/config_test.go internal/account/id.go internal/character/id.go
git commit -m "feat: configure development world login"
```

### Task 2: Authoritative Session State and State-Aware Connection Boundary

**Files:**
- Modify: `internal/engine/network/inbound.go`
- Modify: `internal/engine/network/session.go`
- Modify: `internal/engine/network/connection.go`
- Modify: `internal/engine/network/server.go`
- Modify: `internal/engine/network/connection_test.go`
- Modify: `internal/engine/network/server_test.go`
- Create: `internal/engine/network/session_test.go`

**Interfaces:**
- Consumes: `account.ID`, `character.ID`, existing `Message`, connection queues, and server lifecycle.
- Produces: `SessionState` constants `StateHandshake`, `StateLogin`, `StateGame`, and `StateClosed`.
- Produces: `Identity{AccountID account.ID, CharacterID character.ID}` and `Identity.Valid() bool`.
- Produces: `Session.State() SessionState`, `Session.Identity() (Identity, bool)`, `Session.AcceptHandshake() error`, `Session.Authenticate(Identity) error`, and `Session.Send(Message) error`.
- Produces: `InboundHandler.Handle(*Session, Message) (deliver bool, err error)` and `GameplayMessage` embedding `Message` plus `Gameplay()`.
- Changes: `NewConnection` and `NewServer` accept an `InboundHandler`; `Connection.Session() *Session` exposes the associated application boundary.

- [ ] **Step 1: Write failing state and identity tests**

Test initial `Handshake`, `AcceptHandshake` entering `Login`, repeated acceptance failing with `ErrIllegalSessionState`, authentication outside `Login` failing, invalid identity failing, valid authentication atomically storing both IDs and entering `Game`, and closure preventing later transitions.

```go
func TestSessionAuthenticationStoresIdentityAndEntersGame(t *testing.T) {
	s := newSession(1, 1, nil)
	if err := s.AcceptHandshake(); err != nil { t.Fatal(err) }
	want := Identity{AccountID: 41, CharacterID: 73}
	if err := s.Authenticate(want); err != nil { t.Fatal(err) }
	got, ok := s.Identity()
	if !ok || got != want || s.State() != StateGame { t.Fatalf("state=%v identity=%+v ok=%t", s.State(), got, ok) }
}
```

- [ ] **Step 2: Run the session tests and witness failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run Session`

Expected: build failure because state and identity APIs are absent.

- [ ] **Step 3: Implement session state under one mutex**

Initialize sessions in `StateHandshake`. `AcceptHandshake` permits only `Handshake -> Login`; `Authenticate` validates both IDs then stores the complete identity and enters `Game` while holding the same lock; `close` enters `Closed`; `enqueue` rejects closed sessions before a bounded non-blocking send. `Session.Send` checks state then delegates to `Connection.Send`.

- [ ] **Step 4: Run session tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run Session`

Expected: PASS.

- [ ] **Step 5: Write failing handler-boundary connection tests**

Define a test handler that records calls and a test gameplay message implementing `Gameplay()`. Prove consumed messages are absent from inbound, delivered messages are enqueued, handler errors close the connection, all handler responses use `Session.Send`, and a concurrent `Connection.Close` leaves the session closed with no later enqueue.

- [ ] **Step 6: Run boundary tests and witness failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run 'TestConnection(InboundHandler|ConcurrentClose)'`

Expected: build failure because connection handler support is absent.

- [ ] **Step 7: Integrate the handler without changing writer ownership**

After registry decode, call the handler. Continue reading when it consumes a message; close on handler error; otherwise call the session's bounded enqueue. Set the session outbound delegate to `Connection.Send`. In shutdown, mark the session closed before closing the done signal and transport. Propagate the same handler from `Server` to every accepted connection.

- [ ] **Step 8: Update existing constructor call sites and run race tests**

Pass nil handlers in low-level tests that intentionally test transport/queue mechanics. Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race ./internal/engine/network`

Expected: PASS with no race reports.

- [ ] **Step 9: Commit**

```bash
git add internal/engine/network/inbound.go internal/engine/network/session.go internal/engine/network/connection.go internal/engine/network/server.go internal/engine/network/connection_test.go internal/engine/network/server_test.go internal/engine/network/session_test.go
git commit -m "feat: add state-aware network sessions"
```

### Task 3: Login Packets and Development Validator

**Files:**
- Create: `internal/world/login/packets.go`
- Create: `internal/world/login/codecs.go`
- Create: `internal/world/login/codecs_test.go`
- Create: `internal/world/login/validator.go`
- Create: `internal/world/login/validator_test.go`

**Interfaces:**
- Consumes: generic protocol reader/writer, network registry, and `network.Identity`.
- Produces: opcodes and messages `ClientHello`, `ServerHello`, `LoginRequest`, `LoginAccepted`, and `LoginRejected`.
- Produces: `RegisterCodecs(*network.Registry) error`.
- Produces: `LoginValidator.Validate(string) (network.Identity, error)`.
- Produces: `NewDevelopmentValidator(string, network.Identity) *DevelopmentValidator`.
- Produces: `ErrInvalidTicket` and `ErrValidatorUnavailable`.

- [ ] **Step 1: Write failing codec tests**

Register all login codecs in a real network registry and prove literal round trips for every message. Verify `LoginRequest` encodes only its length-prefixed ticket, an empty ticket decodes structurally, underflow is malformed, and appended identity-like bytes trigger the existing `network.ErrTrailingPayload`.

- [ ] **Step 2: Run codec tests and witness failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login -run Codec`

Expected: build failure because packet and codec APIs are absent.

- [ ] **Step 3: Implement packet types and codec registration**

Assign five package-local production opcodes, implement `Opcode()` for each message, and use only existing reader/writer primitives. Each codec handles exactly its declared fields.

- [ ] **Step 4: Run codec tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login -run Codec`

Expected: PASS.

- [ ] **Step 5: Write failing development-validator tests**

Prove an exact configured opaque ticket returns literal typed IDs; mismatched and empty tickets return `ErrInvalidTicket`; and an empty ticket, invalid identity, or configuration-disabled construction returns `ErrValidatorUnavailable`. Never include ticket values in error assertions or logs.

```go
func TestDevelopmentValidatorMapsTicketToIdentity(t *testing.T) {
	want := network.Identity{AccountID: 41, CharacterID: 73}
	got, err := NewDevelopmentValidator("opaque", want).Validate("opaque")
	if err != nil || got != want { t.Fatalf("identity=%+v err=%v", got, err) }
}
```

- [ ] **Step 6: Run validator tests and witness failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login -run Validator`

Expected: build failure because validator APIs are absent.

- [ ] **Step 7: Implement the development-only validator**

Copy the configured ticket into private bytes, mark the validator available only for a non-empty ticket and valid identity, compare with `subtle.ConstantTimeCompare`, and return only generic sentinel errors.

- [ ] **Step 8: Run all packet and validator tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/world/login/packets.go internal/world/login/codecs.go internal/world/login/codecs_test.go internal/world/login/validator.go internal/world/login/validator_test.go
git commit -m "feat: add world login packets and validator"
```

### Task 4: State-Aware World Login Handler

**Files:**
- Create: `internal/world/login/handler.go`
- Create: `internal/world/login/handler_test.go`

**Interfaces:**
- Consumes: `network.InboundHandler`, session state/identity/send APIs, login packets, `LoginValidator`, and `network.GameplayMessage`.
- Produces: `NewHandler(protocolVersion uint16, validator LoginValidator) *Handler` and `Handler.Handle(*network.Session, network.Message) (bool, error)`.
- Produces: `ErrUnsupportedProtocol`, `ErrIllegalMessageState`, `ErrGameplayBeforeLogin`, and `ErrUnexpectedMessage`.

- [ ] **Step 1: Write failing end-to-end handler tests**

Use `net.Pipe`, a real registry with login and private gameplay codecs, a real connection, and the recording transport where useful. Cover:

- valid hello queues accepted `ServerHello`, moves to `Login`, and leaves inbound empty;
- unsupported version queues rejected `ServerHello` and remains in `Handshake`;
- repeated hello and login-before-handshake close with `ErrIllegalMessageState`;
- valid hello plus configured ticket stores both typed IDs, enters `Game`, queues `LoginAccepted`, and leaves inbound empty;
- invalid, empty, unavailable, and partially configured validator paths queue only `LoginRejected`, retain no identity, and remain in `Login`;
- a marked gameplay message before `Game` closes with `ErrGameplayBeforeLogin`;
- a marked gameplay message in `Game` reaches inbound;
- hello or login after `Game` closes with `ErrIllegalMessageState`;
- an unmarked unknown application message returns `ErrUnexpectedMessage`;
- disconnect closes the session and prevents later handler transitions or enqueues.

- [ ] **Step 2: Run handler tests and witness failure**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/world/login -run Handler`

Expected: build failure because `Handler` and its errors do not exist.

- [ ] **Step 3: Implement the minimal type switch and central state calls**

For `ClientHello`, require `Handshake`, compare versions, queue `ServerHello`, and call `AcceptHandshake` only on a match. For `LoginRequest`, require `Login`; queue `LoginRejected` on validation failure; otherwise call `Authenticate` before queueing `LoginAccepted`. For every other message, require the `GameplayMessage` marker and `Game`, then return `deliver=true`.

- [ ] **Step 4: Run handler tests under race detection**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race ./internal/world/login`

Expected: PASS with no race reports.

- [ ] **Step 5: Commit**

```bash
git add internal/world/login/handler.go internal/world/login/handler_test.go
git commit -m "feat: gate sessions through world login"
```

### Task 5: World Server Composition and Permanent Documentation

**Files:**
- Modify: `cmd/server/main.go`
- Create: `docs/architecture/world-login-handshake.md`

**Interfaces:**
- Consumes: config protocol/development mapping, login registry/validator/handler, and the updated network server constructor.
- Produces no new runtime interface.

- [ ] **Step 1: Wire the world login flow in `cmd/server`**

Register login codecs before starting the listener. Build an unavailable development validator when `Config.DevelopmentLogin` is nil; otherwise convert validated raw config IDs once to `account.ID` and `character.ID`, build `network.Identity`, and construct the development validator. Pass `login.NewHandler(cfg.ProtocolVersion, validator)` into `network.NewServer`.

- [ ] **Step 2: Verify the executable compiles with no login environment**

Run: `env -u WORLD_DEV_LOGIN_TICKET -u WORLD_DEV_ACCOUNT_ID -u WORLD_DEV_CHARACTER_ID GOCACHE=/private/tmp/clearwaste-go-cache go test ./cmd/server`

Expected: PASS; missing development configuration does not prevent startup composition.

- [ ] **Step 3: Write permanent architecture documentation**

Document the packet sequence, state table, config variables and absence/partial behavior, no-ticket-logging rule, generic client failures, gameplay queue boundary, and the exact future replacement: implement `login.LoginValidator` for signed short-lived world tickets and replace only construction in `cmd/server`.

- [ ] **Step 4: Run all tests**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go docs/architecture/world-login-handshake.md
git commit -m "feat: enable world login handshake"
```

### Task 6: Final Verification

**Files:**
- Modify only files required to correct a witnessed verification failure, together with its regression test.

**Interfaces:**
- Consumes all prior task APIs.
- Produces no new feature API.

- [ ] **Step 1: Format changed Go files**

Run: `gofmt -w config cmd/server internal/account internal/character internal/engine/network internal/world/login`

- [ ] **Step 2: Run the full suite freshly**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -count=1 ./...`

Expected: PASS.

- [ ] **Step 3: Run static analysis**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go vet ./...`

Expected: PASS.

- [ ] **Step 4: Run the full race suite freshly**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race -count=1 ./...`

Expected: PASS with no race reports.

- [ ] **Step 5: Audit scope and diff cleanliness**

Run: `git diff --check && git status --short`

Confirm login packets are outside generic protocol, ticket values are absent from logs/session identity, all application responses route through `Send`, and no deferred feature entered the implementation.

- [ ] **Step 6: Commit a verification correction only if one was required**

Stage only the correction and its failing-then-passing regression test. Do not create an empty commit when verification required no change.
