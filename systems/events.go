// filepath: d:\Temp\ebiten-rogue\systems\events.go
package systems

import (
	"ebiten-rogue/ecs"
)

// Event type constants
const (
	EventCollision  ecs.EventType = "collision"
	EventMovement   ecs.EventType = "movement"
	EventCombat     ecs.EventType = "combat"
	EventDeath      ecs.EventType = "death"
	EventItemPickup ecs.EventType = "item_pickup"
)

// CollisionEvent is emitted when entities collide
type CollisionEvent struct {
	EntityID1 ecs.EntityID // First entity involved in collision
	EntityID2 ecs.EntityID // Second entity involved in collision
	X         int          // X position where collision occurred
	Y         int          // Y position where collision occurred
}

// Type returns the event type
func (e CollisionEvent) Type() ecs.EventType {
	return EventCollision
}

// PlayerMoveEvent is emitted when the player moves
type PlayerMoveEvent struct {
	EntityID ecs.EntityID // Entity that moved
	FromX    int          // Starting X position
	FromY    int          // Starting Y position
	ToX      int          // Ending X position
	ToY      int          // Ending Y position
}

// Type returns the event type
func (e PlayerMoveEvent) Type() ecs.EventType {
	return EventMovement
}

// CombatEvent is emitted during combat
type CombatEvent struct {
	AttackerID ecs.EntityID // Entity performing the attack
	DefenderID ecs.EntityID // Entity being attacked
	Damage     int          // Amount of damage dealt
	IsHit      bool         // Whether the attack hit
}

// Type returns the event type
func (e CombatEvent) Type() ecs.EventType {
	return EventCombat
}

// DeathEvent is emitted when an entity dies
type DeathEvent struct {
	EntityID ecs.EntityID // Entity that died
	KillerID ecs.EntityID // Entity that caused the death (if any)
}

// Type returns the event type
func (e DeathEvent) Type() ecs.EventType {
	return EventDeath
}

// ItemPickupEvent is emitted when an entity picks up an item
type ItemPickupEvent struct {
	EntityID ecs.EntityID // Entity picking up the item
	ItemID   ecs.EntityID // Item being picked up
}

// Type returns the event type
func (e ItemPickupEvent) Type() ecs.EventType {
	return EventItemPickup
}

// EntityMoveEvent is emitted when any entity (including AI) moves
type EntityMoveEvent struct {
	EntityID ecs.EntityID // Entity that moved
	FromX    int          // Starting X position
	FromY    int          // Starting Y position
	ToX      int          // Ending X position
	ToY      int          // Ending Y position
}

// Type returns the event type
func (e EntityMoveEvent) Type() ecs.EventType {
	return EventMovement
}
