package network

import "errors"

// ErrInboundBackpressure means a session produced input faster than it was consumed.
var ErrInboundBackpressure = errors.New("inbound queue full")

// InboundHandler applies application protocol policy after message decoding.
// A false deliver result means the handler consumed the message.
type InboundHandler interface {
	Handle(*Session, Message) (deliver bool, err error)
}

// GameplayMessage marks messages eligible for the simulation-facing queue.
type GameplayMessage interface {
	Message
	Gameplay()
}
