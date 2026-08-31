# Gameplay request protocol

Gameplay packets are client intent, not authoritative state. Their path is:

```text
TCP -> frame decoder -> feature codec -> concrete request
    -> login state gate -> bounded Session.Inbound queue
    -> future tick -> snapshot -> Decide -> Apply
```

The network goroutine only validates binary representation. Queueing a request does not move a player, change run state, start an interaction, or acknowledge success. Future simulation processing will validate game rules and produce authoritative outbound updates.

All fixed-width fields use network/big-endian byte order. Payloads must have exactly the documented length; truncation and trailing bytes are protocol errors.

## Opcodes

Stable values live in `internal/engine/network/opcode`:

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

The neutral opcode package owns identifiers only. Login, movement, and interaction packages own their codecs.

## MoveRequest

Exactly one payload byte:

```text
offset  size  field
0       1     direction uint8
```

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

Values 8 through 255 are invalid. The decoder performs no collision, bounds, cooldown, occupancy, traversability, or corner-cutting checks.

## SetRunEnabled

Exactly one payload byte:

```text
offset  size  field
0       1     enabled uint8: 0=false, 1=true
```

Other values are invalid. The decoder does not modify running state or inspect stamina.

## InteractRequest

Exactly nine payload bytes:

```text
offset  size  field
0       8     runtime entity ID uint64
8       1     action uint8: 1=Chop, 2=Mine
```

`entity.ID` identifies one spawned runtime world entity across the network/gameplay boundary. A future ECS may use it directly or map it to a private handle. Zero is permanently reserved as invalid.

```text
AccountID != CharacterID != entity.ID != content definition ID
```

Interaction targets are never names or tree/rock definition IDs. The decoder performs no target existence, kind, range, tool, timer, reward, inventory, XP, or animation checks.

## Session queue

Only messages implementing the marker-only `network.GameplayMessage` reach `Session.Inbound()`, and only after login enters `Game`. The queue uses the existing `WORLD_INBOUND_QUEUE_CAPACITY`. Overflow closes only the offending connection with `ErrInboundBackpressure`; movement coalescing is deferred until tick-consumption semantics exist.

Frame size continues to use `WORLD_MAX_PAYLOAD_BYTES`, and login gating continues to use `WORLD_PROTOCOL_VERSION`. `WORLD_TICK_MS` remains unrelated simulation configuration.
