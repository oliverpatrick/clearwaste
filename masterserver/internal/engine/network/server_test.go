package network

import (
	"errors"
	"io"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/engine/network/transport"
)

func TestConnectionIDsIncreaseWithoutUsingZero(t *testing.T) {
	s := NewServer(newFakeListener(), NewRegistry(), testConnectionConfig(), nil)
	first, err := s.allocateConnectionID()
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.allocateConnectionID()
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 2 {
		t.Fatalf("ids=%d,%d", first, second)
	}
}

func TestConnectionIDWraparoundIsRejected(t *testing.T) {
	s := NewServer(newFakeListener(), NewRegistry(), testConnectionConfig(), nil)
	s.nextConnectionID.Store(math.MaxUint64)
	if _, err := s.allocateConnectionID(); !errors.Is(err, ErrConnectionIDExhausted) {
		t.Fatalf("error=%v", err)
	}
	if _, err := s.allocateConnectionID(); !errors.Is(err, ErrConnectionIDExhausted) {
		t.Fatalf("second error=%v", err)
	}
}

func TestServerTracksSessionsAndRemovesDisconnectedClients(t *testing.T) {
	listener := newFakeListener()
	server := NewServer(listener, newTestRegistry(t), testConnectionConfig(), nil)
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve() }()
	t.Cleanup(func() { _ = server.Close() })

	accepted, peer := net.Pipe()
	t.Cleanup(func() { _ = peer.Close() })
	listener.accepted <- accepted
	waitForSessionCount(t, server, 1)
	if id := server.Sessions()[0].ID(); id != 1 {
		t.Fatalf("session id=%d", id)
	}

	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	waitForSessionCount(t, server, 0)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("serve error=%v", err)
	}
}

func TestServerCloseStopsAllConnections(t *testing.T) {
	listener := newFakeListener()
	server := NewServer(listener, newTestRegistry(t), testConnectionConfig(), nil)
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve() }()

	peers := make([]net.Conn, 0, 2)
	for range 2 {
		accepted, peer := net.Pipe()
		listener.accepted <- accepted
		peers = append(peers, peer)
	}
	waitForSessionCount(t, server, 2)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-serveErr; err != nil {
		t.Fatalf("serve error=%v", err)
	}
	if sessions := server.Sessions(); len(sessions) != 0 {
		t.Fatalf("sessions=%d", len(sessions))
	}
	for _, peer := range peers {
		if _, err := peer.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
			t.Fatalf("peer read error=%v", err)
		}
		_ = peer.Close()
	}
}

func TestServerSendsTypedMessageToSessionConnection(t *testing.T) {
	listener := newFakeListener()
	server := NewServer(listener, newTestRegistry(t), testConnectionConfig(), nil)
	go func() { _ = server.Serve() }()
	t.Cleanup(func() { _ = server.Close() })

	accepted, peer := net.Pipe()
	t.Cleanup(func() { _ = peer.Close() })
	listener.accepted <- accepted
	waitForSessionCount(t, server, 1)
	id := server.Sessions()[0].ID()
	if err := server.Send(id, testMessage{Value: 77}); err != nil {
		t.Fatal(err)
	}
	frame, err := protocol.DecodeFrame(peer, 1024)
	if err != nil {
		t.Fatal(err)
	}
	message, err := newTestRegistry(t).Decode(frame)
	if err != nil {
		t.Fatal(err)
	}
	if got := message.(testMessage).Value; got != 77 {
		t.Fatalf("value=%d", got)
	}
}

func TestServerSendRejectsUnknownSession(t *testing.T) {
	server := NewServer(newFakeListener(), newTestRegistry(t), testConnectionConfig(), nil)
	if err := server.Send(99, testMessage{}); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("error=%v", err)
	}
}

func waitForSessionCount(t *testing.T, server *Server, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(server.Sessions()) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("sessions=%d want=%d", len(server.Sessions()), want)
}

type fakeListener struct {
	accepted chan transport.Conn
	closed   chan struct{}
	once     sync.Once
}

func newFakeListener() *fakeListener {
	return &fakeListener{accepted: make(chan transport.Conn), closed: make(chan struct{})}
}

func (l *fakeListener) Accept() (transport.Conn, error) {
	select {
	case conn := <-l.accepted:
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *fakeListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}
