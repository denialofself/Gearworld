// filepath: d:\Temp\ebiten-rogue\systems\events.go
package systems

import (
	"ebiten-rogue/ecs"
)

// Event type constants
const (
	EventCollision   ecs.EventType = "collision"
	EventMovement    ecs.EventType = "movement"
	EventCombat      ecs.EventType = "combat"
	EventDeath       ecs.EventType = "death"
	EventItemPickup  ecs.EventType = "item_pickup"
	EventEnemyAttack ecs.EventType = "enemy_attack"
	EventRest        ecs.EventType = "rest"
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

// EnemyAttackEvent is emitted when an enemy attacks the player
type EnemyAttackEvent struct {
	AttackerID ecs.EntityID // Enemy entity performing the attack
	TargetID   ecs.EntityID // Player entity being attacked
	X          int          // X position where attack occurred
	Y          int          // Y position where attack occurred
}

// Type returns the event type
func (e EnemyAttackEvent) Type() ecs.EventType {
	return EventEnemyAttack
}

// RestEvent is emitted when an entity rests for a turn
type RestEvent struct {
	EntityID ecs.EntityID // Entity that is resting
}

// Type returns the event type
func (e RestEvent) Type() ecs.EventType {
	return EventRest
}

// StatsChangedEvent is emitted when an entity's stats change
type StatsChangedEvent struct {
	EntityID ecs.EntityID
}

// EntityDeathEvent is emitted when an entity dies
type EntityDeathEvent struct {
	EntityID ecs.EntityID
}

// PlayerMoveAttemptEvent is emitted when a player attempts to move
type PlayerMoveAttemptEvent struct {
	EntityID  ecs.EntityID
	FromX     int
	FromY     int
	ToX       int
	ToY       int
	Direction int
}

// Type returns the event type
func (e PlayerMoveAttemptEvent) Type() ecs.EventType {
	return "player_move_attempt"
}

// TurnCompletedEvent is emitted when a player completes a turn
type TurnCompletedEvent struct {
	EntityID ecs.EntityID
}

// Type returns the event type
func (e TurnCompletedEvent) Type() ecs.EventType {
	return "turn_completed"
}

// ItemEquippedEvent is emitted when an item is equipped
type ItemEquippedEvent struct {
	EntityID ecs.EntityID // Entity that equipped the item
	ItemID   ecs.EntityID // Item that was equipped
	Slot     string       // Slot where the item was equipped
}

// Type returns the event type
func (e ItemEquippedEvent) Type() ecs.EventType {
	return "item_equipped"
}

// ItemUnequippedEvent is emitted when an item is unequipped
type ItemUnequippedEvent struct {
	EntityID ecs.EntityID // Entity that unequipped the item
	ItemID   ecs.EntityID // Item that was unequipped
	Slot     string       // Slot from which the item was unequipped
}

// Type returns the event type
func (e ItemUnequippedEvent) Type() ecs.EventType {
	return "item_unequipped"
}
