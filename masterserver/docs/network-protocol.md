# Network Protocol Conventions

## Transport shape

The server currently targets a custom binary protocol over TCP.

WebSocket support may be added later behind transport abstractions, but should not weaken the typed packet model.

## Frame

Current frame header:

```text
[opcode uint16][payload length uint32][payload bytes...]
```

All integer fields are big-endian unless a packet explicitly documents otherwise.

## Connection model

Each connection should have:

- one reader goroutine;
- one writer goroutine;
- one bounded inbound path;
- one bounded outbound queue.

Avoid multiple goroutines concurrently writing directly to the socket.

## Reader behaviour

The reader should:

1. read the fixed-size header completely;
2. decode opcode and payload length;
3. reject payload lengths above the configured maximum before allocation;
4. allocate/reuse payload storage;
5. read exactly the declared payload;
6. look up the codec for the opcode;
7. decode to a concrete typed message;
8. enqueue the message for simulation/application handling.

Use `io.ReadFull`-style semantics for exact fixed-size reads.

## Protocol errors

Treat these separately:

- unknown opcode;
- malformed payload;
- oversized payload;
- connection I/O failure;
- inbound queue overflow.

Unknown opcode is a protocol violation and should close the connection.

Malformed packet data should not be silently ignored.

## Backpressure

Queues must be bounded.

If a client cannot consume outbound traffic fast enough, the server should apply an explicit backpressure policy rather than allowing unbounded memory growth.

If the bounded server-owned inbound queue is full, closing the offending connection is acceptable for the current design.

## Opcode registry

Opcode values are wire contracts.

Rules:

- values are explicit;
- values remain stable after assignment;
- do not rely on `iota` ordering for a persisted/public wire contract unless constants are explicitly fixed;
- never renumber old opcodes to fill gaps.

Keep one authoritative opcode registry/package.

Feature-owned codecs are encouraged when they keep layout rules close to the owning feature.

## Current gameplay opcode direction

The current design includes or anticipates typed requests such as:

```text
MoveRequest
SetRunEnabled
InteractRequest
```

If existing code already assigns stable opcode values, preserve those exact values.

Do not duplicate opcode constants inside feature packages.

## Current request layouts

Where the current implementation follows these layouts, preserve them unless deliberately versioning the protocol.

```text
MoveRequest
[direction uint8]

SetRunEnabled
[enabled uint8]  // 0 or 1

InteractRequest
[runtime entity ID uint64][action uint8]
```

Direction values:

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

Interaction action values currently intended:

```text
1 Chop
2 Mine
```

A zero runtime target entity ID is invalid.

## Typed decode boundary

Prefer:

```go
type MoveRequest struct {
    Direction Direction
}
```

over generic envelopes such as:

```go
map[string]any
```

or:

```go
type GameplayMessage struct {
    Type string
    Data []byte
}
```

when a concrete typed request is known.

The decoder/registry may expose a small common interface for routing, but feature code should receive concrete types.

## Encoding guidance

For packet-specific payloads:

- use fixed-width integers where practical;
- validate enums/ranges while decoding;
- reject trailing/insufficient bytes unless the packet explicitly allows extension fields;
- keep encode/decode symmetrical;
- write round-trip tests.

Strings should use an explicit length encoding and maximum size.

Never trust a client-declared length without bounds checking.

## Protocol tests

Each codec should normally have:

- successful encode/decode round-trip;
- minimum/maximum valid values;
- invalid enum tests;
- truncated payload tests;
- trailing payload test if exact length is required.

Frame tests should include:

- header round-trip;
- oversized length rejection;
- unknown opcode;
- partial reads;
- multiple sequential frames.
