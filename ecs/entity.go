package ecs

import "sync/atomic"

// EntityID is a unique identifier for an entity
type EntityID uint64

var nextEntityID uint64 = 0

// NewEntityID generates a new unique entity ID
func NewEntityID() EntityID {
	return EntityID(atomic.AddUint64(&nextEntityID, 1))
}

// Entity represents a game object in the ECS architecture
type Entity struct {
	ID EntityID
	// Tags can be used for quick identification (e.g., "player", "enemy")
	Tags map[string]bool
}

// NewEntity creates a new entity
func NewEntity() *Entity {
	return &Entity{
		ID:   NewEntityID(),
		Tags: make(map[string]bool),
	}
}

// AddTag adds a tag to the entity
func (e *Entity) AddTag(tag string) {
	e.Tags[tag] = true
}

// HasTag checks if the entity has a specific tag
func (e *Entity) HasTag(tag string) bool {
	return e.Tags[tag]
}

// RemoveTag removes a tag from the entity
func (e *Entity) RemoveTag(tag string) {
	delete(e.Tags, tag)
}
