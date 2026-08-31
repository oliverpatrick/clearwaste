package transport

import "io"

// Conn is a readable, writable, closable byte stream.
type Conn interface {
	io.Reader
	io.Writer
	io.Closer
}

// Listener accepts byte-stream connections.
type Listener interface {
	Accept() (Conn, error)
	Close() error
}
