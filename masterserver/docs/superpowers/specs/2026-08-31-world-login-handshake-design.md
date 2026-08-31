# World Login and Handshake Design

## Scope

Add the minimum world-server handshake and development login needed to turn a
TCP connection into an authenticated gameplay session. The change introduces
no OAuth, account service, database, character selection, world selection,
ECS spawning, persistence, or gameplay implementation.

The permanent seam is:

```text
opaque ticket -> LoginValidator -> typed AccountID + CharacterID
```

A future signed world-ticket validator replaces only the validator
implementation. Packet decoding, the login handler, session state, and the
gameplay queue contract remain unchanged.

## Package Ownership

- `internal/engine/network/protocol` remains generic framing and binary IO.
- `internal/engine/network` owns connection lifecycle, session protocol state,
  typed identity storage, state-aware inbound handling, queues, and the sole
  writer goroutine.
- `internal/account` and `internal/character` define small strongly typed IDs.
- `internal/world/login` owns login packet definitions/codecs, login policy,
  the validator interface, and the development-only validator.
- `config` reads protocol and development-login configuration. Login code does
  not call `os.Getenv` or read dotenv files.

No ticket value is logged or stored in a session.

## State Model

`Session` is authoritative for protocol state and owns these states:

```text
Handshake -> Login -> Game -> Closed
     |          |       |
     +----------+-------+-> Closed
```

A new session starts in `Handshake`; there is no redundant transient
`Connected` state. `Connection.shutdown` moves the session to `Closed`, while
the connection's done channel remains only a goroutine-lifecycle signal.

Session methods perform all state mutation under a mutex:

- `Transition(Handshake, Login)` accepts a valid client hello;
- `Authenticate(identity)` atomically verifies `Login`, stores the full typed
  identity, and enters `Game`;
- `close()` enters `Closed` from any state;
- transitions after `Closed` fail.

Authentication never stores a partial identity. Failed validation leaves both
identity fields unset and the session in `Login`.

## Inbound Boundary

The existing read path becomes:

```text
transport -> frame decode -> message decode -> InboundHandler
                                             |-> login consumed
                                             `-> gameplay queue
```

`network.InboundHandler` has one operation that receives a session and typed
message and returns whether the message may be delivered. The connection still
owns queue insertion and backpressure.

`network.GameplayMessage` explicitly marks messages eligible for the future
simulation boundary. `world/login.Handler` applies these rules centrally:

- `ClientHello` is legal only in `Handshake` and is always consumed;
- `LoginRequest` is legal only in `Login` and is always consumed;
- a marked gameplay message is delivered only in `Game`;
- every other message/state combination is a typed protocol-state error.

Before enqueueing, the session rechecks that it is not closed. This prevents a
concurrent shutdown from permitting a later transition or gameplay enqueue.
Login messages can never appear in `Session.Inbound()`.

The handler sends responses with `Session.Send`, which delegates to the
existing bounded connection outbound queue. It never writes to the transport;
only the connection writer goroutine encodes frames and writes bytes.

## Packets and Codecs

Login packet definitions and codec registration live in
`internal/world/login`, not the generic protocol package. The initial packet
set is:

- `ClientHello { ProtocolVersion uint16 }`;
- `ServerHello { Accepted bool, ProtocolVersion uint16 }`;
- `LoginRequest { Ticket string }`;
- `LoginAccepted {}`;
- `LoginRejected {}`.

The ticket is the only login-request field and uses the existing `uint16`
length-prefixed string codec. The client never supplies account or character
identity. The registry's trailing-byte check rejects attempts to append fields.

A supported hello transitions to `Login` and queues an accepted
`ServerHello`. An unsupported version queues a generic rejected `ServerHello`
and remains in `Handshake`. A valid ticket atomically authenticates the session
and queues `LoginAccepted`. Invalid, empty, unavailable, or mismatched tickets
queue `LoginRejected`, retain no identity, and remain in `Login` so no
credential-validation detail is exposed.

Malformed packets, unknown opcodes, login before handshake, repeated hello or
login in an illegal state, and gameplay before `Game` are protocol violations
and close the connection through existing error handling.

## Validator Seam

```go
type LoginValidator interface {
	Validate(ticket string) (network.Identity, error)
}
```

`network.Identity` contains only `account.ID` and `character.ID`.

`DevelopmentValidator` is explicitly development-only. It holds one configured
opaque ticket and one configured identity. It compares ticket bytes without
logging them and returns the configured identity only on an exact, non-empty
match. An unconfigured validator rejects all tickets. It performs no database
or external service access.

## Configuration

`config.Config` gains:

- `ProtocolVersion uint16`, read from `WORLD_PROTOCOL_VERSION`, default `1`;
- optional `DevelopmentLogin`, sourced from `WORLD_DEV_LOGIN_TICKET`,
  `WORLD_DEV_ACCOUNT_ID`, and `WORLD_DEV_CHARACTER_ID`.

The development ticket and IDs have no source defaults. The existing
`.env.example` remains unchanged.

Configuration behavior is:

- all three valid values: construct the development validator mapping;
- all absent: validator is unavailable and rejects every login;
- only some present: validator is unavailable and rejects every login;
- any supplied ID that is malformed or zero: `config.Load` fails;
- an empty ticket counts as absent.

The command builds domain IDs from the validated configuration and never
passes raw client identity into the session.

## Errors and Client Exposure

Internal sentinel errors distinguish unsupported protocol version, illegal
state/message, gameplay before login, invalid ticket, unavailable validator,
invalid identity, closed session, malformed payload, and disconnect.

Client responses expose only acceptance or rejection. They contain no ticket,
configured identity, validation reason, or implementation detail.

## Tests

Tests cover:

- configuration defaults, complete mapping, absent/partial mapping, and
  malformed/zero IDs;
- login packet codec round trips and malformed/trailing payloads;
- initial `Handshake` state and legal/illegal transitions;
- successful authentication atomically storing typed identity;
- invalid authentication storing no identity;
- development validator valid, invalid, empty, partial, and unavailable cases;
- valid and unsupported hello behavior;
- login before handshake and repeated hello rejection;
- successful hello plus login entering `Game`;
- gameplay before `Game` rejection;
- login/hello after `Game` rejection;
- login messages never entering the gameplay queue;
- responses using the existing outbound writer path;
- disconnect cleanup and concurrent shutdown blocking transitions/enqueueing.

Final verification runs `go test ./...`, `go vet ./...`, and
`go test -race ./...`.

## Deferred Work

OAuth, passwords, account service calls, signed world tickets, expiration and
replay protection, databases, Redis, character/world selection, multiple
worlds, build identity, `WorldID`, ECS spawning, persistence, movement, and
combat remain deferred until their owning systems exist.
