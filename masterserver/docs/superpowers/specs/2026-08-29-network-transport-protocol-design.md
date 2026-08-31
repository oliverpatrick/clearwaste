# TCP Transport and Binary Protocol Foundation

## Scope

Implement only the byte transport, binary protocol, connection lifecycle, and
per-session message boundaries. Gameplay validation, ECS mutation,
authentication, persistence, and gameplay opcodes remain outside this work.

## Wire Format

Frames use network byte order (big endian):

| Offset | Size | Field |
|---:|---:|---|
| 0 | 2 | opcode (`uint16`) |
| 2 | 4 | payload length (`uint32`) |
| 6 | N | payload bytes |

The decoder reads the header and payload with `io.ReadFull`. This supports
fragmented reads and multiple frames in one TCP stream without maintaining a
second accumulation buffer. It rejects a declared payload length above the
configured maximum before allocating the payload. A clean disconnect before
the next header is `io.EOF`; a partial header or payload is
`io.ErrUnexpectedEOF`. The decoder preserves that distinction for logging.

Normal EOF, truncated input, oversized payloads, malformed payloads, and
unknown opcodes remain distinguishable errors. An unknown opcode is a protocol
violation and closes the connection.

## Protocol Primitives and Codecs

The protocol package provides a bounds-checked `Reader` and an append-based
`Writer` for fixed-width integers, booleans, and length-prefixed byte/string
values. Readers return underflow errors instead of panicking. Strings and byte
slices use a `uint16` length prefix, accept zero-length values, and reject
values that cannot fit.

A registry associates an opcode with typed encode and decode functions. Go
requires type erasure to store heterogeneous codecs; it remains private to the
registry. Callers register and consume typed messages rather than passing
`any` through application code. No production gameplay opcodes are introduced;
tests use private message types and opcodes.

## Transport Boundary

The transport package exposes only small connection and listener interfaces.
TCP implementations wrap `net.Conn` and `net.Listener`; higher layers do not
depend on those concrete types. This is sufficient for another byte-oriented
transport later without implementing WebSocket support now.

`network.Connection` is distinct from `transport.Conn`: the former owns a
session, protocol processing, queues, goroutines, and shutdown; the latter is
only a readable, writable, closable byte stream.

## Sessions and Data Flow

Each accepted connection gets a `Session` containing only a strongly typed
`ConnectionID` and a bounded inbound queue:

```text
transport -> reader goroutine -> decode -> session inbound queue
transport <- writer goroutine <- encode <- connection outbound queue
```

The future simulation will drain each session's inbound queue at the start of
a tick and associate the session with gameplay identity when authentication is
implemented. The network package never mutates ECS or gameplay state.

The queue initially preserves all message order and has no gameplay-specific
coalescing. A full inbound queue closes that connection so one noisy client
cannot consume another session's capacity.

## Connection Ownership and Backpressure

Each connection has exactly one reader goroutine and one writer goroutine.
Only the writer goroutine writes to the transport, so no socket write mutex is
needed. Shutdown is idempotent: closing the transport unblocks reads/writes,
signals both goroutines, and allows the server to remove the connection.

Outbound sends are non-blocking. If the bounded outbound queue is full,
`Send` returns a typed backpressure error and closes the slow connection. No
arbitrary authoritative messages are dropped while leaving a desynchronized
client connected.

## Server and Configuration

The server listens on the existing `Config.TCPAddress`, accepts TCP
connections, assigns monotonically increasing connection IDs with an atomic
`uint64` counter, owns the active connection registry, and shuts all
connections down when stopped. ID zero is reserved. If incrementing the
counter wraps to zero, allocation returns a typed exhaustion error and the
newly accepted connection is closed rather than reusing an ID.

The existing environment-backed configuration is extended consistently with
only values required by this implementation:

- maximum protocol payload size;
- per-session inbound queue capacity;
- per-connection outbound queue capacity.

No speculative buffer sizes, deadlines, keepalive periods, worker counts, or
connection limits are added.

## Tests

Tests will be written before implementation and will cover:

- frame encode/decode round trip;
- fragmented input;
- multiple frames in one stream;
- clean EOF versus truncated header/payload errors;
- malformed frame lengths;
- oversized payload rejection before allocation;
- primitive reader/writer round trips, valid empty strings/bytes, and reader
  underflow;
- typed registry round trip and unknown opcode rejection;
- clean connection shutdown;
- inbound overflow disconnect;
- outbound overflow disconnect;
- single-writer behavior where practical;
- connection ID allocation and wraparound rejection;
- configuration defaults, environment overrides, and invalid limits.

Final verification runs `go test ./...`, `go vet ./...`, and
`go test -race ./...`.

## Deferred Work

WebSocket support, authentication and identity fields, gameplay opcodes,
message coalescing, priority queues, ECS integration, movement, gathering,
combat, inventory, content loading, and persistence are intentionally deferred
until their owning features exist.
