package network

import "errors"

var (
	ErrOutboundBackpressure = errors.New("outbound queue full")
	ErrConnectionClosed     = errors.New("connection closed")
)
