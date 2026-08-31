// Package entity defines runtime entity identity crossing gameplay boundaries.
package entity

// ID identifies one spawned runtime world entity, independently of ECS handles.
type ID uint64

// Invalid is reserved and never identifies a runtime entity.
const Invalid ID = 0

// Valid reports whether the ID may identify a runtime entity.
func (id ID) Valid() bool { return id != Invalid }
