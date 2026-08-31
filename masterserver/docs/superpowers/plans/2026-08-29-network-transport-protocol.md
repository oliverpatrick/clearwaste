# Network Transport and Protocol Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a bounded TCP connection layer and safe big-endian binary protocol that delivers typed messages to per-session queues without touching gameplay or ECS state.

**Architecture:** TCP is hidden behind small byte-stream interfaces. One reader and one writer goroutine own each connection; codecs turn framed payloads into typed messages held by a bounded per-session queue, while queue overflow closes only the offending connection.

**Tech Stack:** Go 1.25 standard library only (`encoding/binary`, `io`, `net`, `sync`, `sync/atomic`, `testing`).

**Spec:** `docs/superpowers/specs/2026-08-29-network-transport-protocol-design.md`

## Global Constraints

- Wire headers are exactly six bytes: big-endian `uint16` opcode followed by big-endian `uint32` payload length.
- Reject payload lengths above the configured maximum before allocation.
- Preserve `io.EOF` for a clean boundary disconnect and `io.ErrUnexpectedEOF` for partial frames.
- Zero-length `uint16`-prefixed strings and byte slices are valid.
- Unknown opcodes, malformed payloads, and either queue overflowing close that connection.
- Only the writer goroutine writes to a connection's transport.
- Keep `Session` limited to `ConnectionID` and its inbound queue.
- Add no gameplay opcodes, gameplay validation, ECS references, WebSocket implementation, dependencies, or speculative network settings.

## File Map

- `config/config.go`: existing environment-backed configuration plus the three required network limits.
- `config/config_test.go`: network limit defaults, overrides, and validation.
- `internal/engine/network/protocol/errors.go`: stable protocol sentinel errors.
- `internal/engine/network/protocol/opcode.go`: `Opcode` definition.
- `internal/engine/network/protocol/frame.go`: `Frame` and six-byte header constant.
- `internal/engine/network/protocol/reader.go`: bounds-checked primitive payload reader.
- `internal/engine/network/protocol/writer.go`: append-based primitive payload writer.
- `internal/engine/network/protocol/decoder.go`: stream frame decoder.
- `internal/engine/network/protocol/encoder.go`: complete frame writer.
- `internal/engine/network/transport/transport.go`: minimal connection/listener interfaces.
- `internal/engine/network/transport/tcp.go`: TCP implementations.
- `internal/engine/network/transport/websocket.go`: delete the empty, uncompilable file.
- `internal/engine/network/registry.go`: typed message codec registry with erased storage confined internally.
- `internal/engine/network/inbound.go`: inbound backpressure error.
- `internal/engine/network/outbound.go`: backpressure sentinel error.
- `internal/engine/network/session.go`: connection ID and bounded inbound queue.
- `internal/engine/network/connection.go`: two-goroutine lifecycle, decoding, encoding, and shutdown.
- `internal/engine/network/server.go`: accept loop, connection IDs, active sessions, and shutdown.
- `cmd/server/main.go`: load existing config, listen on `TCPAddress`, and stop on process signal.

---

### Task 1: Required Network Configuration

**Files:**
- Modify: `config/config.go`
- Modify: `config/config_test.go`

**Interfaces:**
- Produces: `Config.MaxPayloadSize uint32`, `Config.InboundQueueCapacity int`, and `Config.OutboundQueueCapacity int`.
- Produces environment variables: `WORLD_MAX_PAYLOAD_BYTES`, `WORLD_INBOUND_QUEUE_CAPACITY`, and `WORLD_OUTBOUND_QUEUE_CAPACITY`.

- [ ] **Step 1: Write failing configuration tests**

Add table-driven tests proving defaults of 64 KiB, 64 inbound messages, and 256 outbound messages; environment overrides; rejection of zero/negative capacities; and rejection of payload values outside `uint32`.

```go
func TestDefaultNetworkConfiguration(t *testing.T) {
	clearNetworkEnv(t)
	cfg, err := Load()
	if err != nil { t.Fatal(err) }
	if cfg.MaxPayloadSize != 64<<10 || cfg.InboundQueueCapacity != 64 || cfg.OutboundQueueCapacity != 256 {
		t.Fatalf("network config=%+v", cfg)
	}
}
```

- [ ] **Step 2: Verify the tests fail for missing fields**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./config`

Expected: build failure because the three `Config` fields do not exist.

- [ ] **Step 3: Implement minimal environment-backed limits**

Use the existing `envInt` convention for capacities and a small `envUint32` helper using `strconv.ParseUint`. Validate every configured value is positive. Do not add unrelated settings.

- [ ] **Step 4: Verify configuration tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./config`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add config/config.go config/config_test.go
git commit -m "feat: configure network protocol limits"
```

### Task 2: Binary Payload Reader and Writer

**Files:**
- Create: `internal/engine/network/protocol/errors.go`
- Modify: `internal/engine/network/protocol/opcode.go`
- Modify: `internal/engine/network/protocol/reader.go`
- Modify: `internal/engine/network/protocol/writer.go`
- Create: `internal/engine/network/protocol/reader_writer_test.go`

**Interfaces:**
- Produces: `protocol.Opcode uint16`, `protocol.NewReader([]byte) *Reader`, primitive reader methods returning `(value, error)`, and `Reader.Remaining() int`.
- Produces: `protocol.NewWriter(capacity int) *Writer`, append-only primitive writer methods, `Writer.Bytes([]byte) error`, `Writer.String(string) error`, and `Writer.Buffer() []byte`.
- Produces: `ErrUnderflow`, `ErrInvalidBool`, and `ErrValueTooLarge`.

- [ ] **Step 1: Write failing primitive and edge-case tests**

Test all requested integer widths, signed `Int32`, booleans, strings, and byte slices in one round trip. Add separate tests for empty string/bytes, underflow, invalid boolean bytes, and values longer than `math.MaxUint16`.

```go
func TestEmptyLengthPrefixedValuesAreValid(t *testing.T) {
	w := NewWriter(4)
	if err := w.String(""); err != nil { t.Fatal(err) }
	if err := w.Bytes(nil); err != nil { t.Fatal(err) }
	r := NewReader(w.Buffer())
	s, err := r.String()
	if err != nil || s != "" { t.Fatalf("string=%q err=%v", s, err) }
	b, err := r.Bytes()
	if err != nil || len(b) != 0 { t.Fatalf("bytes=%v err=%v", b, err) }
}
```

- [ ] **Step 2: Verify tests fail against the declaration-only skeleton**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/protocol -run 'Test(Primitive|Empty|Reader|Invalid|Oversized)'`

Expected: build failure because the declared methods have no bodies or required constructors/errors are absent.

- [ ] **Step 3: Implement the minimal reader and writer**

Use `encoding/binary.BigEndian`, explicit bounds checks before slicing, and `append` into one writer buffer. Copy length-prefixed bytes out of the reader so decoded messages do not retain a whole frame accidentally.

- [ ] **Step 4: Verify primitive tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/protocol -run 'Test(Primitive|Empty|Reader|Invalid|Oversized)'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/network/protocol/errors.go internal/engine/network/protocol/opcode.go internal/engine/network/protocol/reader.go internal/engine/network/protocol/writer.go internal/engine/network/protocol/reader_writer_test.go
git commit -m "feat: add binary payload reader and writer"
```

### Task 3: Binary Frame Streaming

**Files:**
- Modify: `internal/engine/network/protocol/frame.go`
- Modify: `internal/engine/network/protocol/decoder.go`
- Modify: `internal/engine/network/protocol/encoder.go`
- Create: `internal/engine/network/protocol/frame_test.go`

**Interfaces:**
- Produces: `const HeaderSize = 6` and `Frame{Opcode Opcode, Payload []byte}`.
- Produces: `DecodeFrame(io.Reader, uint32) (Frame, error)`.
- Produces: `EncodeFrame(io.Writer, Frame, uint32) error`.
- Produces: `ErrPayloadTooLarge` and `ErrInvalidFrame`.

- [ ] **Step 1: Write failing frame tests**

Cover round trip, a reader that returns one byte per call, two concatenated frames, clean empty-stream EOF, partial header, partial payload, maximum-size acceptance, and oversized length rejection. The oversized test supplies only a six-byte header to prove rejection occurs before payload allocation/read.

```go
func TestDecodeFrameDistinguishesEOFAndTruncation(t *testing.T) {
	if _, err := DecodeFrame(bytes.NewReader(nil), 1024); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream error=%v", err)
	}
	if _, err := DecodeFrame(bytes.NewReader([]byte{0}), 1024); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial header error=%v", err)
	}
}
```

- [ ] **Step 2: Verify frame tests fail**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/protocol -run 'Test(Encode|Decode|Fragmented|Multiple|Oversized)'`

Expected: build failure because frame streaming functions are absent.

- [ ] **Step 3: Implement frame encoding and decoding**

Build the header in a `[HeaderSize]byte`, check `uint32(len(payload))` safely, reject the decoded length before `make`, use `io.ReadFull` for both reads, and use a local `writeFull` loop so short writes cannot truncate a frame.

- [ ] **Step 4: Verify all protocol tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/protocol`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/network/protocol/frame.go internal/engine/network/protocol/decoder.go internal/engine/network/protocol/encoder.go internal/engine/network/protocol/frame_test.go
git commit -m "feat: add binary frame streaming"
```

### Task 4: Typed Codec Registry

**Files:**
- Modify: `internal/engine/network/registry.go`
- Create: `internal/engine/network/registry_test.go`

**Interfaces:**
- Produces: `Message` with `Opcode() protocol.Opcode`.
- Produces: `NewRegistry() *Registry`.
- Produces: generic `Register[T Message](*Registry, protocol.Opcode, decode func(*protocol.Reader) (T, error), encode func(*protocol.Writer, T) error) error`.
- Produces: `Registry.Decode(protocol.Frame) (Message, error)` and `Registry.Encode(Message) (protocol.Frame, error)`.
- Produces: `ErrUnknownOpcode`, `ErrDuplicateOpcode`, `ErrWrongMessageType`, and `ErrTrailingPayload`.

- [ ] **Step 1: Write failing registry tests with private messages**

Define a test-only typed message and codec. Prove typed round trip, duplicate registration rejection, unknown opcode rejection, wrong outbound type rejection, and rejection when a decoder leaves bytes unread.

```go
type testMessage struct{ Value uint32 }
func (testMessage) Opcode() protocol.Opcode { return 1 }

func TestRegistryRoundTrip(t *testing.T) {
	r := NewRegistry()
	if err := Register(r, 1, decodeTestMessage, encodeTestMessage); err != nil { t.Fatal(err) }
	frame, err := r.Encode(testMessage{Value: 42})
	if err != nil { t.Fatal(err) }
	got, err := r.Decode(frame)
	if err != nil { t.Fatal(err) }
	if got != (testMessage{Value: 42}) { t.Fatalf("message=%v", got) }
}
```

- [ ] **Step 2: Verify registry tests fail**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run Registry`

Expected: build failure because the registry API does not exist.

- [ ] **Step 3: Implement the registry**

Store non-generic closure pairs in the private map; generic `Register` performs the only message type assertion. Guard registration and lookup with `sync.RWMutex` so codecs can be registered during setup and read concurrently by connections.

- [ ] **Step 4: Verify registry tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run Registry`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/network/registry.go internal/engine/network/registry_test.go
git commit -m "feat: add typed packet codec registry"
```

### Task 5: TCP Transport Boundary

**Files:**
- Modify: `internal/engine/network/transport/transport.go`
- Modify: `internal/engine/network/transport/tcp.go`
- Delete: `internal/engine/network/transport/websocket.go`
- Create: `internal/engine/network/transport/tcp_test.go`

**Interfaces:**
- Produces: `Conn` embedding `io.Reader`, `io.Writer`, and `io.Closer`.
- Produces: `Listener` with `Accept() (Conn, error)` and `Close() error`.
- Produces: `ListenTCP(address string) (Listener, error)`.

- [ ] **Step 1: Write a failing TCP boundary test**

Listen on `127.0.0.1:0`, accept through the transport interface, exchange a short byte slice with a standard-library TCP client, and close both ends.

- [ ] **Step 2: Verify the transport test fails**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/transport`

Expected: build failure because `tcp.go` and `websocket.go` are empty and the interfaces are incomplete.

- [ ] **Step 3: Implement the minimal TCP wrappers**

Wrap `net.Listener.Accept` so it returns `transport.Conn`. Do not expose `net.Conn`, add socket tuning, or implement WebSocket.

- [ ] **Step 4: Verify transport tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network/transport`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/network/transport/transport.go internal/engine/network/transport/tcp.go internal/engine/network/transport/tcp_test.go
git rm internal/engine/network/transport/websocket.go
git commit -m "feat: add TCP transport boundary"
```

### Task 6: Per-Session Connection Lifecycle and Backpressure

**Files:**
- Modify: `internal/engine/network/inbound.go`
- Modify: `internal/engine/network/outbound.go`
- Modify: `internal/engine/network/session.go`
- Modify: `internal/engine/network/connection.go`
- Create: `internal/engine/network/connection_test.go`

**Interfaces:**
- Produces: `ConnectionID uint64`.
- Produces: `Session.ID() ConnectionID` and `Session.Inbound() <-chan Message`.
- Produces: `ConnectionConfig{MaxPayloadSize uint32, InboundQueueCapacity int, OutboundQueueCapacity int}`.
- Produces: `NewConnection(ConnectionID, transport.Conn, *Registry, ConnectionConfig) *Connection`.
- Produces: `Connection.Start()`, `Connection.Send(Message) error`, `Connection.Close() error`, `Connection.Done() <-chan struct{}`, and `Connection.Wait() error`.
- Produces: `ErrInboundBackpressure`, `ErrOutboundBackpressure`, and `ErrConnectionClosed`.

- [ ] **Step 1: Write failing lifecycle and inbound tests**

Use `net.Pipe`, which satisfies `transport.Conn`, and the private registry codec. Prove an encoded fragmented frame reaches only that connection's session, clean peer close yields `io.EOF`, malformed input closes the connection with the typed error, explicit close terminates both goroutines, and inbound overflow closes only that connection.

- [ ] **Step 2: Verify lifecycle/inbound tests fail**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run 'TestConnection(Reads|Close|Inbound|Malformed)'`

Expected: build failure because connection lifecycle APIs are absent.

- [ ] **Step 3: Implement session and the read path**

Create the bounded session channel in `NewConnection`. `Start` launches one reader and one writer. The reader calls `protocol.DecodeFrame`, then `Registry.Decode`, then performs a non-blocking send to its session. On full inbound capacity, record `ErrInboundBackpressure` and close the transport.

- [ ] **Step 4: Verify read-path tests pass**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run 'TestConnection(Reads|Close|Inbound|Malformed)'`

Expected: PASS.

- [ ] **Step 5: Write failing outbound and single-writer tests**

Use a blocking test transport whose `Write` records concurrent entry. Prove ordered encoded frames are written by one goroutine and that filling the outbound queue returns `ErrOutboundBackpressure` and closes the connection.

- [ ] **Step 6: Verify outbound tests fail**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run 'TestConnection(Outbound|SingleWriter)'`

Expected: FAIL because outbound queue behavior is not implemented.

- [ ] **Step 7: Implement the writer path and idempotent shutdown**

`Send` performs a non-blocking channel send without ever closing the outbound channel. The writer alone calls `Registry.Encode` and `protocol.EncodeFrame`. A `sync.Once` closes the done signal and transport; a `WaitGroup` makes `Wait` leak-free. First terminal error storage remains synchronized and retains `io.EOF` versus truncation.

- [ ] **Step 8: Verify all connection tests pass under race detection**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race ./internal/engine/network`

Expected: PASS with no race reports.

- [ ] **Step 9: Commit**

```bash
git add internal/engine/network/inbound.go internal/engine/network/outbound.go internal/engine/network/session.go internal/engine/network/connection.go internal/engine/network/connection_test.go
git commit -m "feat: add bounded session connections"
```

### Task 7: Server Lifecycle, Connection IDs, and Executable Wiring

**Files:**
- Modify: `internal/engine/network/server.go`
- Create: `internal/engine/network/server_test.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `NewServer(transport.Listener, *Registry, ConnectionConfig) *Server`.
- Produces: `Server.Serve() error`, `Server.Close() error`, and `Server.Sessions() []*Session`.
- Produces: `ErrConnectionIDExhausted`.

- [ ] **Step 1: Write failing ID and server lifecycle tests**

Use a fake listener backed by channels. Prove IDs begin at one and increase, setting the internal atomic counter to `math.MaxUint64` makes the next allocation return `ErrConnectionIDExhausted`, accepted connections appear in `Sessions`, disconnected connections disappear, and server close terminates the accept loop and all connections.

```go
func TestConnectionIDWraparoundIsRejected(t *testing.T) {
	s := NewServer(newFakeListener(), NewRegistry(), testConnectionConfig())
	s.nextConnectionID.Store(math.MaxUint64)
	if _, err := s.allocateConnectionID(); !errors.Is(err, ErrConnectionIDExhausted) {
		t.Fatalf("error=%v", err)
	}
}
```

- [ ] **Step 2: Verify server tests fail**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/engine/network -run 'Test(Server|ConnectionID)'`

Expected: build failure because the server APIs are absent.

- [ ] **Step 3: Implement the accept loop and active registry**

Use `atomic.Uint64` for IDs and reserve zero. Protect the active connection map with one mutex. Each accepted connection is registered before start and removed after `Wait`; server close closes the listener, snapshots connections under lock, then closes and waits outside the lock.

- [ ] **Step 4: Verify server tests pass under race detection**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race ./internal/engine/network -run 'Test(Server|ConnectionID)'`

Expected: PASS with no race reports.

- [ ] **Step 5: Wire `cmd/server` to existing configuration**

Load `config.Load`, create an empty codec registry, call `transport.ListenTCP(cfg.TCPAddress)`, pass the three configured limits to `network.NewServer`, and use `signal.NotifyContext` to close the server on `SIGINT`/`SIGTERM`. Treat listener closure during requested shutdown as normal.

- [ ] **Step 6: Verify all packages compile and test**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./...`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/engine/network/server.go internal/engine/network/server_test.go cmd/server/main.go
git commit -m "feat: serve bounded TCP sessions"
```

### Task 8: Final Requirements and Concurrency Verification

**Files:**
- Modify only files required to correct failures found by verification.

**Interfaces:**
- Consumes all prior task APIs.
- Produces no new feature API.

- [ ] **Step 1: Format production and test Go files**

Run: `gofmt -w config cmd/server internal/engine/network`

- [ ] **Step 2: Run the full test suite**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test ./...`

Expected: PASS.

- [ ] **Step 3: Run static analysis**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go vet ./...`

Expected: PASS.

- [ ] **Step 4: Run all tests with race detection**

Run: `GOCACHE=/private/tmp/clearwaste-go-cache go test -race ./...`

Expected: PASS with no race reports.

- [ ] **Step 5: Check formatting and the final diff**

Run: `git diff --check && git status --short`

Expected: no whitespace errors; status contains only intentional project files.

- [ ] **Step 6: Commit any verification-only correction**

If verification required a correction, stage only that correction and its regression test, then commit it with a message naming the corrected behavior. If no correction was needed, do not create an empty commit.
