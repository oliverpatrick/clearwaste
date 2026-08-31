package network

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"master/clearwaste/internal/engine/network/protocol"
)

func TestConnectionReadsFragmentedFrameIntoItsSession(t *testing.T) {
	connection, peer := startTestConnection(t, 1, testConnectionConfig())
	frame, err := newTestRegistry(t).Encode(testMessage{Value: 42})
	if err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := protocol.EncodeFrame(&wire, frame, 1024); err != nil {
		t.Fatal(err)
	}
	writeErr := make(chan error, 1)
	go func() {
		for _, b := range wire.Bytes() {
			if _, err := peer.Write([]byte{b}); err != nil {
				writeErr <- err
				return
			}
		}
		writeErr <- nil
	}()

	select {
	case got := <-connection.session.Inbound():
		message, ok := got.(testMessage)
		if !ok || message.Value != 42 {
			t.Fatalf("message=%T %+v", got, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for inbound message")
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
}

func TestConnectionCleanPeerCloseReturnsEOF(t *testing.T) {
	connection, peer := startTestConnection(t, 1, testConnectionConfig())
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := connection.Wait(); !errors.Is(err, io.EOF) {
		t.Fatalf("error=%v", err)
	}
}

func TestConnectionCloseStopsGoroutines(t *testing.T) {
	connection, _ := startTestConnection(t, 1, testConnectionConfig())
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := connection.Wait(); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("error=%v", err)
	}
	select {
	case <-connection.Done():
	default:
		t.Fatal("connection done signal remains open")
	}
}

func TestConnectionMalformedFrameClosesWithProtocolError(t *testing.T) {
	connection, peer := startTestConnection(t, 1, testConnectionConfig())
	if err := protocol.EncodeFrame(peer, protocol.Frame{Opcode: 99}, 1024); err != nil {
		t.Fatal(err)
	}
	if err := connection.Wait(); !errors.Is(err, ErrUnknownOpcode) {
		t.Fatalf("error=%v", err)
	}
}

func TestConnectionInboundOverflowClosesOnlyNoisySession(t *testing.T) {
	config := testConnectionConfig()
	config.InboundQueueCapacity = 1
	noisy, noisyPeer := startTestConnection(t, 1, config)
	healthy, healthyPeer := startTestConnection(t, 2, config)

	writeTestMessage(t, noisyPeer, 1)
	writeTestMessage(t, noisyPeer, 2)
	if err := noisy.Wait(); !errors.Is(err, ErrInboundBackpressure) {
		t.Fatalf("noisy error=%v", err)
	}

	writeTestMessage(t, healthyPeer, 3)
	select {
	case got := <-healthy.session.Inbound():
		if got.(testMessage).Value != 3 {
			t.Fatalf("healthy message=%+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("healthy session did not receive its message")
	}
	select {
	case <-healthy.Done():
		t.Fatal("healthy connection was closed")
	default:
	}
}

func TestConnectionOutboundOverflowClosesSlowClient(t *testing.T) {
	transport := newBlockingConn()
	config := testConnectionConfig()
	config.OutboundQueueCapacity = 1
	connection := NewConnection(1, transport, newTestRegistry(t), config, nil)
	connection.Start()

	if err := connection.Send(testMessage{Value: 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-transport.writeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not start")
	}
	if err := connection.Send(testMessage{Value: 2}); err != nil {
		t.Fatal(err)
	}
	if err := connection.Send(testMessage{Value: 3}); !errors.Is(err, ErrOutboundBackpressure) {
		t.Fatalf("send error=%v", err)
	}
	if err := connection.Wait(); !errors.Is(err, ErrOutboundBackpressure) {
		t.Fatalf("connection error=%v", err)
	}
}

func TestConnectionSingleWriterPreservesOutboundOrder(t *testing.T) {
	transport := newRecordingConn()
	connection := NewConnection(1, transport, newTestRegistry(t), testConnectionConfig(), nil)
	connection.Start()
	t.Cleanup(func() {
		_ = connection.Close()
		_ = connection.Wait()
	})
	if err := connection.Send(testMessage{Value: 10}); err != nil {
		t.Fatal(err)
	}
	if err := connection.Send(testMessage{Value: 20}); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for transport.size() < 20 {
		select {
		case <-transport.wrote:
		case <-deadline:
			t.Fatalf("wire length=%d", transport.size())
		}
	}
	if transport.maxWriters.Load() != 1 {
		t.Fatalf("concurrent writers=%d", transport.maxWriters.Load())
	}
	wire := bytes.NewReader(transport.bytes())
	for _, want := range []uint32{10, 20} {
		frame, err := protocol.DecodeFrame(wire, 1024)
		if err != nil {
			t.Fatal(err)
		}
		message, err := newTestRegistry(t).Decode(frame)
		if err != nil {
			t.Fatal(err)
		}
		if got := message.(testMessage).Value; got != want {
			t.Fatalf("value=%d want=%d", got, want)
		}
	}
}

func TestConnectionInboundHandlerConsumesMessage(t *testing.T) {
	called := make(chan struct{}, 1)
	handler := testInboundHandler(func(*Session, Message) (bool, error) {
		called <- struct{}{}
		return false, nil
	})
	connection, peer := startHandlerTestConnection(t, handler)
	writeTestMessage(t, peer, 1)
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("handler not called")
	}
	select {
	case message := <-connection.Session().Inbound():
		t.Fatalf("consumed message reached inbound: %+v", message)
	default:
	}
}

func TestConnectionInboundHandlerDeliversMessage(t *testing.T) {
	handler := testInboundHandler(func(*Session, Message) (bool, error) { return true, nil })
	connection, peer := startHandlerTestConnection(t, handler)
	writeTestMessage(t, peer, 7)
	select {
	case message := <-connection.Session().Inbound():
		if message.(testMessage).Value != 7 {
			t.Fatalf("message=%+v", message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("message not delivered")
	}
}

func TestConnectionInboundHandlerResponseUsesOutboundWriter(t *testing.T) {
	handler := testInboundHandler(func(session *Session, _ Message) (bool, error) {
		return false, session.Send(testMessage{Value: 99})
	})
	_, peer := startHandlerTestConnection(t, handler)
	writeTestMessage(t, peer, 1)
	frame, err := protocol.DecodeFrame(peer, 1024)
	if err != nil {
		t.Fatal(err)
	}
	message, err := newTestRegistry(t).Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if message.(testMessage).Value != 99 {
		t.Fatalf("response=%+v", message)
	}
}

func TestConnectionInboundHandlerErrorClosesConnection(t *testing.T) {
	want := errors.New("handler rejected message")
	handler := testInboundHandler(func(*Session, Message) (bool, error) { return false, want })
	connection, peer := startHandlerTestConnection(t, handler)
	writeTestMessage(t, peer, 1)
	if err := connection.Wait(); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
}

func TestConnectionConcurrentClosePreventsHandlerEnqueue(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handler := testInboundHandler(func(*Session, Message) (bool, error) {
		close(started)
		<-release
		return true, nil
	})
	connection, peer := startHandlerTestConnection(t, handler)
	writeDone := make(chan struct{})
	go func() {
		_, _ = peer.Write([]byte{0, 1, 0, 0, 0, 4, 0, 0, 0, 1})
		close(writeDone)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler not called")
	}
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	<-writeDone
	if err := connection.Wait(); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("error=%v", err)
	}
	if connection.Session().State() != StateClosed {
		t.Fatalf("state=%v", connection.Session().State())
	}
	if message, ok := <-connection.Session().Inbound(); ok {
		t.Fatalf("message enqueued after close: %+v", message)
	}
}

func startTestConnection(t *testing.T, id ConnectionID, config ConnectionConfig) (*Connection, net.Conn) {
	t.Helper()
	server, peer := net.Pipe()
	connection := NewConnection(id, server, newTestRegistry(t), config, nil)
	connection.Start()
	t.Cleanup(func() {
		_ = peer.Close()
		_ = connection.Close()
		_ = connection.Wait()
	})
	return connection, peer
}

func startHandlerTestConnection(t *testing.T, handler InboundHandler) (*Connection, net.Conn) {
	t.Helper()
	server, peer := net.Pipe()
	connection := NewConnection(1, server, newTestRegistry(t), testConnectionConfig(), handler)
	connection.Start()
	t.Cleanup(func() {
		_ = peer.Close()
		_ = connection.Close()
		_ = connection.Wait()
	})
	return connection, peer
}

func testConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxPayloadSize:        1024,
		InboundQueueCapacity:  4,
		OutboundQueueCapacity: 4,
	}
}

func writeTestMessage(t *testing.T, peer io.Writer, value uint32) {
	t.Helper()
	frame, err := newTestRegistry(t).Encode(testMessage{Value: value})
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.EncodeFrame(peer, frame, 1024); err != nil {
		t.Fatal(err)
	}
}

type testInboundHandler func(*Session, Message) (bool, error)

func (h testInboundHandler) Handle(session *Session, message Message) (bool, error) {
	return h(session, message)
}

type blockingConn struct {
	closed       chan struct{}
	closeOnce    sync.Once
	writeStarted chan struct{}
}

func newBlockingConn() *blockingConn {
	return &blockingConn{closed: make(chan struct{}), writeStarted: make(chan struct{}, 1)}
}

func (c *blockingConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, io.EOF
}

func (c *blockingConn) Write([]byte) (int, error) {
	select {
	case c.writeStarted <- struct{}{}:
	default:
	}
	<-c.closed
	return 0, io.ErrClosedPipe
}

func (c *blockingConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

type recordingConn struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	closed     chan struct{}
	closeOnce  sync.Once
	wrote      chan struct{}
	writers    atomic.Int32
	maxWriters atomic.Int32
}

func newRecordingConn() *recordingConn {
	return &recordingConn{closed: make(chan struct{}), wrote: make(chan struct{}, 8)}
}

func (c *recordingConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, io.EOF
}

func (c *recordingConn) Write(p []byte) (int, error) {
	writers := c.writers.Add(1)
	for {
		max := c.maxWriters.Load()
		if writers <= max || c.maxWriters.CompareAndSwap(max, writers) {
			break
		}
	}
	defer c.writers.Add(-1)
	c.mu.Lock()
	n, err := c.buf.Write(p)
	c.mu.Unlock()
	select {
	case c.wrote <- struct{}{}:
	default:
	}
	return n, err
}

func (c *recordingConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

func (c *recordingConn) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Len()
}

func (c *recordingConn) bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf.Bytes()...)
}
