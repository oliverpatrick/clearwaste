package transport

import "net"

type tcpListener struct {
	listener net.Listener
}

// ListenTCP listens for TCP byte streams at address.
func ListenTCP(address string) (Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &tcpListener{listener: listener}, nil
}

func (l *tcpListener) Accept() (Conn, error) { return l.listener.Accept() }

func (l *tcpListener) Close() error { return l.listener.Close() }

func (l *tcpListener) Addr() net.Addr { return l.listener.Addr() }
