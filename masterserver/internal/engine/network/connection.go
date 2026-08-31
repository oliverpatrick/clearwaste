package network

import (
	"sync"

	"master/clearwaste/internal/engine/network/protocol"
	"master/clearwaste/internal/engine/network/transport"
)

// ConnectionConfig contains the bounded resources required by one connection.
type ConnectionConfig struct {
	MaxPayloadSize        uint32
	InboundQueueCapacity  int
	OutboundQueueCapacity int
}

// Connection owns one transport and its reader and writer goroutines.
type Connection struct {
	transport transport.Conn
	registry  *Registry
	config    ConnectionConfig
	handler   InboundHandler
	session   *Session
	outbound  chan Message
	done      chan struct{}

	startOnce sync.Once
	closeOnce sync.Once
	wg        sync.WaitGroup
	errMu     sync.Mutex
	err       error
}

// NewConnection constructs a stopped connection.
func NewConnection(id ConnectionID, conn transport.Conn, registry *Registry, config ConnectionConfig, handler InboundHandler) *Connection {
	connection := &Connection{
		transport: conn,
		registry:  registry,
		config:    config,
		handler:   handler,
		session:   newSession(id, config.InboundQueueCapacity),
		outbound:  make(chan Message, config.OutboundQueueCapacity),
		done:      make(chan struct{}),
	}
	connection.session.setSender(connection.Send)
	return connection
}

// Start launches the connection's sole reader and writer.
func (c *Connection) Start() {
	c.startOnce.Do(func() {
		c.wg.Add(2)
		go c.readLoop()
		go c.writeLoop()
	})
}

// Send queues one typed outbound message without blocking the caller.
// A full queue closes the connection because dropping authoritative packets
// would leave a slow client desynchronized.
func (c *Connection) Send(message Message) error {
	select {
	case <-c.done:
		return ErrConnectionClosed
	default:
	}
	select {
	case <-c.done:
		return ErrConnectionClosed
	case c.outbound <- message:
		return nil
	default:
		c.shutdown(ErrOutboundBackpressure)
		return ErrOutboundBackpressure
	}
}

// Close shuts down the connection. It is safe to call more than once.
func (c *Connection) Close() error { return c.shutdown(ErrConnectionClosed) }

// Done is closed when connection shutdown begins.
func (c *Connection) Done() <-chan struct{} { return c.done }

// Session returns the application-facing state and message boundary.
func (c *Connection) Session() *Session { return c.session }

// Wait waits for connection goroutines and returns the first terminal reason.
func (c *Connection) Wait() error {
	c.wg.Wait()
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.err
}

func (c *Connection) readLoop() {
	defer c.wg.Done()
	defer close(c.session.inbound)
	for {
		frame, err := protocol.DecodeFrame(c.transport, c.config.MaxPayloadSize)
		if err != nil {
			c.shutdown(err)
			return
		}
		message, err := c.registry.Decode(frame)
		if err != nil {
			c.shutdown(err)
			return
		}
		if c.handler != nil {
			deliver, err := c.handler.Handle(c.session, message)
			if err != nil {
				c.shutdown(err)
				return
			}
			if !deliver {
				continue
			}
		}
		if err := c.session.enqueue(message); err != nil {
			c.shutdown(err)
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case <-c.done:
			return
		case message := <-c.outbound:
			frame, err := c.registry.Encode(message)
			if err != nil {
				c.shutdown(err)
				return
			}
			if err := protocol.EncodeFrame(c.transport, frame, c.config.MaxPayloadSize); err != nil {
				c.shutdown(err)
				return
			}
		}
	}
}

func (c *Connection) shutdown(reason error) error {
	var closeErr error
	c.closeOnce.Do(func() {
		c.session.close()
		c.errMu.Lock()
		c.err = reason
		c.errMu.Unlock()
		close(c.done)
		closeErr = c.transport.Close()
	})
	return closeErr
}
