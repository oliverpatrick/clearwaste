# Network Protocol

## Purpose

The project uses a custom binary protocol between clients and game/account servers.

The protocol should remain:

* compact
* explicitly typed
* deterministic
* stable once packet layouts are released
* independent of Godot
* independent of internal ECS implementation details

The protocol communicates application/runtime state.

It must not expose implementation-specific structures such as raw ECS memory or Godot scene paths.

---

# Transport

TCP is the current transport.

WebSocket support may be added later but should not influence the core packet model prematurely.

The protocol layer should be transport-independent where practical.

---

# Frame format

Current packet framing is:

```text
[opcode uint16][payload length uint32][payload bytes]
```

All integer values are big-endian unless a packet explicitly documents otherwise.

Header size:

```text
opcode         2 bytes
payload length 4 bytes
----------------------
total header   6 bytes
```

Example:

```text
00 06
00 00 00 01
02
```

represents:

```text
opcode  = 6
length  = 1
payload = 02
```

---

# Packet reading

Packet decoding should:

1. read the complete fixed-size header
2. decode opcode
3. decode payload size
4. reject oversized payloads before allocation
5. allocate/reuse an appropriate payload buffer
6. read the complete payload
7. dispatch to the codec registered for the opcode

Use full reads rather than assuming one network read returns an entire packet.

In Go this means using behaviour equivalent to:

```go
io.ReadFull(...)
```

for both header and payload.

TCP is a byte stream, not a message transport.

---

# Packet size limits

A maximum payload size must exist.

Never trust the payload length supplied by a remote connection.

Reject packets that exceed the configured maximum before allocating memory.

This protects the server from malformed or malicious frames attempting extreme allocations.

---

# Opcodes

Opcodes are stable wire identifiers.

Do not use `iota` for wire values when stability matters unless explicit assignments guarantee compatibility.

Prefer:

```go
const (
    OpcodeLoginRequest   Opcode = 1
    OpcodeLoginResponse  Opcode = 2
    ...
)
```

rather than allowing values to shift when constants are reordered.

---

# Current opcode allocation

The currently agreed gameplay additions are:

```text
1–5 existing login/session packets

6 MoveRequest

7 SetRunEnabled

8 InteractRequest
```

Existing values must be preserved.

Do not renumber existing opcodes to make an enum look cleaner.

---

# Unknown opcodes

An unknown opcode is a protocol violation.

Current policy:

```text
unknown opcode
    ↓
close connection
```

Do not silently ignore unknown gameplay packets.

This makes client/server version errors and malformed traffic easier to detect.

---

# Decode errors

Distinguish at least:

```text
unknown opcode
malformed payload
oversized payload
connection/read failure
```

These are different failure classes.

Tests should make those distinctions explicit.

---

# Codec registry

Packets should be registered by opcode.

Conceptually:

```text
Opcode
    ↓
Codec
    ↓
typed message
```

For example:

```go
type Codec[T any] interface {
    Decode([]byte) (T, error)
    Encode(T) ([]byte, error)
}
```

The exact generic/interface design may differ from this example.

The key rule is that the protocol layer produces concrete typed messages rather than unstructured byte/string maps.

---

# Gameplay messages

Network decoding should produce typed gameplay requests.

For example:

```text
binary bytes
    ↓
MoveRequest
    ↓
gameplay input queue
    ↓
simulation
```

Avoid introducing an unnecessary generic packet-to-request mapping layer when the decoded packet already represents the gameplay request.

Prefer:

```go
MoveRequest
SetRunEnabled
InteractRequest
```

over:

```go
map[string]any
GenericGameplayEnvelope
RawPacket
```

throughout gameplay code.

---

# MoveRequest

Opcode:

```text
6
```

Payload:

```text
[direction uint8]
```

Payload length:

```text
1 byte
```

Directions:

```text
0 North
1 NorthEast
2 East
3 SouthEast
4 South
5 SouthWest
6 West
7 NorthWest
```

Any other direction value is invalid.

Conceptually:

```go
type MoveRequest struct {
    Direction Direction
}
```

This request represents player intent.

The client does not directly set the authoritative player position.

The server validates and applies movement.

---

# SetRunEnabled

Opcode:

```text
7
```

Payload:

```text
[enabled uint8]
```

Payload length:

```text
1 byte
```

Valid values:

```text
0 false
1 true
```

Other values should be rejected as malformed unless the wire definition is intentionally changed.

Conceptually:

```go
type SetRunEnabled struct {
    Enabled bool
}
```

---

# InteractRequest

Opcode:

```text
8
```

Payload:

```text
[target runtime entity ID uint64][action uint8]
```

Payload length:

```text
9 bytes
```

Runtime target ID must not be zero.

Current interaction actions include:

```text
1 Chop
2 Mine
```

Conceptually:

```go
type InteractRequest struct {
    Target entity.ID
    Action InteractionAction
}
```

The target is a runtime entity ID rather than a content definition ID.

For example:

```text
tree definition ID = 50
tree runtime EntityID = 92822
```

The interaction targets:

```text
92822
```

because the player is interacting with one specific runtime tree.

---

# Runtime IDs vs definition IDs

Do not confuse:

```text
definition ID
runtime entity ID
```

Definition ID answers:

```text
what kind of thing is this?
```

Runtime Entity ID answers:

```text
which actual thing in the current world?
```

For example:

```text
NPCDefinitionID = 100
EntityID        = 78129
```

Packets dealing with specific currently spawned entities generally need Entity IDs.

Packets describing what type of entity was spawned may additionally contain definition IDs.

---

# Typed ID sizes

Current interaction target IDs use:

```text
uint64
```

for runtime entity identity.

Do not shrink or change this wire type without an intentional protocol version change.

Internal Go types may wrap primitive numeric types for type safety.

For example:

```go
type ID uint64
```

---

# Server connection model

Current intended Go connection model:

```text
one reader goroutine per connection
one writer goroutine per connection
```

Reader:

```text
TCP
 ↓
frame decoder
 ↓
typed message
 ↓
bounded inbound queue
```

Writer:

```text
game/server
 ↓
bounded outbound queue
 ↓
writer goroutine
 ↓
TCP
```

Do not allow arbitrary gameplay goroutines to write directly to the socket concurrently.

---

# Inbound queue

Inbound messages should enter a server-owned bounded queue.

The queue must not grow without limit.

Current policy:

```text
inbound queue full
    ↓
client cannot keep up / is misbehaving
    ↓
close connection
```

Exact queue capacity should be configuration/implementation driven rather than part of the wire protocol.

---

# Outbound queue

Each connection has a bounded outbound queue.

If the queue is full, return/report a backpressure error rather than silently growing memory usage indefinitely.

The game/server layer can then decide whether to:

* drop nonessential updates where supported
* disconnect the client
* handle backpressure according to packet semantics

Do not block the entire simulation tick indefinitely on one slow client.

---

# Simulation boundary

Network goroutines must not mutate ECS/game state directly.

Preferred flow:

```text
network reader
    ↓
decode packet
    ↓
typed GameplayMessage
    ↓
server inbound queue
    ↓
simulation tick/input collection
    ↓
validated game state mutation
```

This preserves the simulation's concurrency model.

---

# Client authority

Client packets represent requests or intent.

Examples:

```text
MoveRequest
InteractRequest
SetRunEnabled
```

The server remains authoritative.

A movement request does not mean:

```text
player has moved
```

It means:

```text
player wishes to move
```

Likewise:

```text
InteractRequest(target tree, Chop)
```

does not mean the player automatically receives logs.

The server validates:

* target existence
* distance
* line/collision rules
* action compatibility
* player state
* cooldowns
* tool requirements
* other gameplay rules

---

# Server-to-client replication

Server-to-client packets should communicate the subset of authoritative world state required by the client.

Likely future categories include:

```text
entity spawn
entity despawn
entity movement
animation
graphic/effect
ground-item spawn
ground-item remove
inventory update
equipment update
skill update
chat
combat state
object state
```

Do not transmit complete content definitions every time an entity spawns.

Prefer:

```text
EntityID
DefinitionID
position
runtime state
```

and allow the client to resolve static presentation metadata from its local content registry.

---

# Example spawn model

Conceptually:

```text
SpawnNPC

EntityID:     817263
DefinitionID: 100
X:            3200
Y:            3210
Plane:        0
```

The client uses:

```text
DefinitionID 100
    ↓
client content registry
    ↓
asset/visual binding
```

Static names/assets do not need to be repeated over the network.

---

# Protocol ownership

Protocol framing belongs to infrastructure.

Gameplay interpretation belongs to feature packages.

Conceptually:

```text
engine/network
    framing
    connection
    queues
    codecs

game/movement
    meaning of MoveRequest

game/interaction
    meaning of InteractRequest
```

Do not place combat/movement rules in packet decoder code.

---

# Client/server symmetry

Where practical, the client and server may mirror protocol concepts.

For example:

```text
server:
    opcode.go
    decoder.go
    encoder.go

client:
    opcode.gd
    packet_reader.gd
    packet_writer.gd
```

But both must implement the same engine-independent wire specification.

The protocol must not be defined by Godot source code.

---

# Protocol source of truth

Long term, consider defining stable protocol constants/layout metadata in an engine-independent source.

Possible future structure:

```text
content_data/protocol/
```

or:

```text
protocol/
```

The source could later generate:

```text
Go opcode declarations
Godot opcode declarations
protocol documentation
```

Do not introduce generation until the current manually maintained protocol becomes difficult to keep synchronized.

---

# Compatibility rule

Treat established packet layouts as compatibility contracts.

Changing:

```text
uint32 → uint64
field order
opcode value
endianness
```

is a protocol change.

Do not make such changes as incidental refactors.

If protocol versioning becomes necessary, introduce it explicitly.

---

# Tests

Protocol tests should cover at least:

* correct opcode encoding
* correct big-endian byte order
* exact payload lengths
* round-trip encode/decode
* truncated header
* truncated payload
* oversized payload
* unknown opcode
* invalid enum values
* zero/invalid IDs where prohibited
* queue/backpressure behaviour where appropriate

Golden byte tests are useful for stable packets.

Example:

```text
MoveRequest NorthEast

expected bytes:
00 06 00 00 00 01 01
```

This helps detect accidental wire-format changes.

---

# Design principle

The wire protocol should answer:

```text
what does the client want?
what authoritative state does the client need to know?
```

It should not expose:

```text
how the ECS is stored
how Godot scenes are structured
how persistence files are laid out
how internal packages are implemented
```
