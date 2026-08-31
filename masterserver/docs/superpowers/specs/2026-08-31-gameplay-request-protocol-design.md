# Gameplay Request Protocol Design

## Scope

Add the first authenticated gameplay request surface without adding simulation or ECS behaviour:

- move in one of eight Chebyshev directions;
- set run mode explicitly;
- request Chop or Mine against a runtime entity.

Networking validates only binary representation and enum membership. It does not mutate position, running state, entities, collision, resources, inventory, animations, or other gameplay state.

## Data flow

```text
TCP stream
  -> binary frame decoder
  -> registered feature codec
  -> concrete typed gameplay request
  -> state-aware login inbound handler
  -> bounded Session.Inbound queue
  -> future tick drain / Snapshot / Decide / Apply
```

`MoveRequest`, `SetRunEnabled`, and `InteractRequest` implement the existing marker-only `network.GameplayMessage` boundary. The login handler admits them only while the session is in `Game`. Login and handshake packets remain consumed by the login handler and never enter the gameplay queue.

No packet DTO-to-request conversion layer or generic gameplay envelope is introduced. The decoded concrete type is the request later consumed at the tick boundary.

## Opcode contract

Stable application opcodes live in the neutral `internal/engine/network/opcode` package. That package contains only typed constants and their uniqueness contract; it owns no codecs, handlers, or gameplay logic.

Existing login values do not change:

| Value | Message |
| ---: | --- |
| 1 | ClientHello |
| 2 | ServerHello |
| 3 | LoginRequest |
| 4 | LoginAccepted |
| 5 | LoginRejected |
| 6 | MoveRequest |
| 7 | SetRunEnabled |
| 8 | InteractRequest |

All values are explicit rather than `iota`-derived so inserting a future constant cannot renumber the wire protocol. Feature codecs refer to these constants from their owning packages.

## Runtime entity identity

`internal/game/entity.ID` is a `uint64` network-visible runtime entity identifier. It identifies one spawned entity instance in a world and is the canonical identity used across the network/gameplay boundary. A future ECS may use it directly or map it to a private internal handle.

The distinctions are intentional:

```text
AccountID != CharacterID != entity.ID != content definition ID
```

Entity ID `0` is permanently reserved as invalid/unspecified. Interaction packets contain neither display names such as `tree` or `rock` nor content-definition IDs.

## Movement

`internal/game/movement` owns `MoveRequest`, `SetRunEnabled`, their enums, and their codecs.

### MoveRequest

Payload length is exactly one byte:

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 1 | direction `uint8` |

Direction values are:

| Value | Direction |
| ---: | --- |
| 0 | North |
| 1 | NorthEast |
| 2 | East |
| 3 | SouthEast |
| 4 | South |
| 5 | SouthWest |
| 6 | West |
| 7 | NorthWest |

Zero is valid. Values `8` through `255`, truncated payloads, and trailing payload bytes are protocol errors. Decoding does not check collision, bounds, destination occupancy, traversability, cooldown, or diagonal corner cutting.

### SetRunEnabled

Payload length is exactly one byte:

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 1 | enabled: `0=false`, `1=true` |

Values other than `0` and `1`, truncated payloads, and trailing payload bytes are protocol errors. Decoding does not modify running state or inspect stamina.

## Environmental interaction

`internal/game/interaction` owns `InteractRequest`, the supported action enum, and its codec.

Payload length is exactly nine bytes in network/big-endian order:

| Offset | Size | Field |
| ---: | ---: | --- |
| 0 | 8 | runtime `entity.ID` as `uint64` |
| 8 | 1 | action `uint8` |

Actions are:

| Value | Action |
| ---: | --- |
| 0 | invalid/unspecified |
| 1 | Chop |
| 2 | Mine |

Target zero, action zero, actions above two, truncated payloads, and trailing payload bytes are protocol errors. Decoding does not check target existence, range, entity kind, tools, timers, success, resources, XP, inventory, or animations.

## Length validation and errors

Feature decoders read exactly the fixed fields. The existing registry rejects any unread trailing bytes with `network.ErrTrailingPayload`; primitive readers reject truncated payloads with `protocol.ErrUnderflow`, and `Reader.Bool` accepts only `0` or `1`.

Invalid direction, target ID, and action values return feature-owned sentinel errors. As with existing malformed or unknown packets, a decode error closes only that connection.

## Queue and outbound behaviour

All valid authenticated requests use the existing bounded per-session inbound channel configured by `WORLD_INBOUND_QUEUE_CAPACITY`. No new queue, event bus, coalescing, or rate limiter is added.

When the queue is full, the existing `ErrInboundBackpressure` policy closes only the offending connection. Movement inputs remain ordered for now; latest-input replacement can be added when tick-consumption semantics exist.

No acknowledgements are sent. Entering the queue means only that a request was structurally valid and received, not that a movement or interaction succeeded. Future Decide/Apply processing will produce authoritative outbound state.

## Configuration

No configuration is added. The implementation reuses:

- `WORLD_MAX_PAYLOAD_BYTES` through the existing frame decoder;
- `WORLD_INBOUND_QUEUE_CAPACITY` through the existing session queue;
- `WORLD_PROTOCOL_VERSION` and existing login/session state gating.

`WORLD_TICK_MS`, currently defaulting to 600 ms, remains simulation configuration and is not used by packet decoding.

## Registration

`cmd/server` registers login, movement, and interaction codecs before opening the listener. The generic framing package continues to know only about fixed-width binary primitives, frames, and typed opcode values.

## Tests

Tests cover:

- exact opcode values and uniqueness across login and gameplay;
- all eight directions, invalid directions, truncation, and trailing bytes;
- run bytes zero and one, invalid bool bytes, truncation, and trailing bytes;
- Chop and Mine, invalid action values, zero target, truncation, and trailing bytes;
- authenticated delivery of all three concrete requests;
- rejection before `Game`;
- inbound overflow closing only the offending connection;
- absence of gameplay/ECS side effects by keeping feature packages limited to immutable request values and codecs;
- unchanged login packet values and login flow;
- repository-wide tests, vet, and race detection.

## Deferred work

ECS storage, tick draining, snapshot construction, Decide/Apply, collision, pathfinding, movement cooldowns, stamina, chopping/mining mechanics, target lookup, rewards, inventory, animations, authoritative updates, coalescing, and rate limiting remain out of scope.
