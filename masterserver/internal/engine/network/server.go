package network

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"master/clearwaste/internal/engine/network/transport"
)

var (
	// ErrConnectionIDExhausted means every non-zero uint64 connection ID was used.
	ErrConnectionIDExhausted = errors.New("connection ID exhausted")
	// ErrSessionNotFound means a connection ID has no active session.
	ErrSessionNotFound = errors.New("session not found")
)

// Server accepts transports and owns their active connection lifetimes.
type Server struct {
	listener transport.Listener
	registry *Registry
	config   ConnectionConfig
	handler  InboundHandler
	done     chan struct{}

	nextConnectionID atomic.Uint64
	closeOnce        sync.Once
	mu               sync.Mutex
	connections      map[ConnectionID]*Connection
	wg               sync.WaitGroup
}

// NewServer returns a stopped server for listener.
func NewServer(listener transport.Listener, registry *Registry, config ConnectionConfig, handler InboundHandler) *Server {
	return &Server{
		listener:    listener,
		registry:    registry,
		config:      config,
		handler:     handler,
		done:        make(chan struct{}),
		connections: make(map[ConnectionID]*Connection),
	}
}

// Serve accepts connections until the listener fails or the server closes.
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			default:
				return err
			}
		}
		id, err := s.allocateConnectionID()
		if err != nil {
			_ = conn.Close()
			return err
		}
		connection := NewConnection(id, conn, s.registry, s.config, s.handler)

		s.mu.Lock()
		select {
		case <-s.done:
			s.mu.Unlock()
			_ = connection.Close()
			return nil
		default:
		}
		s.connections[id] = connection
		s.wg.Add(1)
		s.mu.Unlock()

		connection.Start()
		go s.waitForConnection(connection)
	}
}

// Sessions returns a snapshot of active application-facing sessions.
func (s *Server) Sessions() []*Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions := make([]*Session, 0, len(s.connections))
	for _, connection := range s.connections {
		sessions = append(sessions, connection.session)
	}
	return sessions
}

// Send queues a typed message for one active session's connection.
func (s *Server) Send(id ConnectionID, message Message) error {
	s.mu.Lock()
	connection, ok := s.connections[id]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %d", ErrSessionNotFound, id)
	}
	return connection.Send(message)
}

// Close stops accepting clients, closes active connections, and waits for cleanup.
func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.done)
		closeErr = s.listener.Close()

		s.mu.Lock()
		connections := make([]*Connection, 0, len(s.connections))
		for _, connection := range s.connections {
			connections = append(connections, connection)
		}
		s.mu.Unlock()
		for _, connection := range connections {
			_ = connection.Close()
		}
		s.wg.Wait()
	})
	return closeErr
}

func (s *Server) allocateConnectionID() (ConnectionID, error) {
	for {
		current := s.nextConnectionID.Load()
		if current == math.MaxUint64 {
			return 0, ErrConnectionIDExhausted
		}
		if s.nextConnectionID.CompareAndSwap(current, current+1) {
			return ConnectionID(current + 1), nil
		}
	}
}

func (s *Server) waitForConnection(connection *Connection) {
	defer s.wg.Done()
	_ = connection.Wait()
	s.mu.Lock()
	delete(s.connections, connection.session.ID())
	s.mu.Unlock()
}
