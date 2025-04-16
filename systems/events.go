// filepath: d:\Temp\ebiten-rogue\systems\events.go
package systems

import (
	"ebiten-rogue/ecs"
)

// Event type constants
const (
	EventCollision         ecs.EventType = "collision"
	EventMovement          ecs.EventType = "movement"
	EventCombat            ecs.EventType = "combat"
	EventDeath             ecs.EventType = "death"
	EventItemPickup        ecs.EventType = "item_pickup"
	EventEnemyAttack       ecs.EventType = "enemy_attack"
	EventRest              ecs.EventType = "rest"
	EventEffects           ecs.EventType = "effects"
	EventEquipItem         ecs.EventType = "equip_item"
	EventUnequipItem       ecs.EventType = "unequip_item"
	EventEquipmentQuery    ecs.EventType = "equipment_query"
	EventEquipmentResponse ecs.EventType = "equipment_response"
	EventCameraUpdate      ecs.EventType = "camera_update"
	EventInventoryUI       ecs.EventType = "inventory_ui"
	EventExamine           ecs.EventType = "examine"
)

// Effect type constants
const (
	EffectTypeHeal        string = "heal"
	EffectTypeDamage      string = "damage"
	EffectTypeStatBoost   string = "stat_boost"
	EffectTypeFOVModify   string = "fov_modify"
	EffectTypeLightSource string = "light_source"
	EffectTypeTemporary   string = "temporary" // For future use with timed effects
)

// EffectsEvent is emitted when an effect should be applied to an entity
type EffectsEvent struct {
	EntityID    ecs.EntityID // Entity to apply the effect to
	EffectType  string       // Type of effect (heal, damage, etc.)
	Property    string       // Property to affect (Health, Attack, etc.)
	Value       interface{}  // Value of the effect
	Source      string       // Source of the effect (item_id, rest, etc.)
	DisplayText string       // Optional text to display
}

// Type returns the event type
func (e EffectsEvent) Type() ecs.EventType {
	return EventEffects
}

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

// EquipItemRequestEvent is emitted when an item should be equipped
type EquipItemRequestEvent struct {
	EntityID ecs.EntityID // Entity to equip the item to
	ItemID   ecs.EntityID // Item to be equipped
	SlotHint string       // Optional slot hint, empty for auto-assignment
}

// Type returns the event type
func (e EquipItemRequestEvent) Type() ecs.EventType {
	return EventEquipItem
}

// UnequipItemRequestEvent is emitted when an item should be unequipped
type UnequipItemRequestEvent struct {
	EntityID ecs.EntityID // Entity to unequip the item from
	ItemID   ecs.EntityID // Item to be unequipped
}

// Type returns the event type
func (e UnequipItemRequestEvent) Type() ecs.EventType {
	return EventUnequipItem
}

// EquipmentQueryRequestEvent is emitted to check equipment status
type EquipmentQueryRequestEvent struct {
	EntityID ecs.EntityID // Entity to check
	ItemID   ecs.EntityID // Item to check
	QueryID  string       // Unique ID to match with response
}

// Type returns the event type
func (e EquipmentQueryRequestEvent) Type() ecs.EventType {
	return EventEquipmentQuery
}

// EquipmentQueryResponseEvent is the response to an equipment query
type EquipmentQueryResponseEvent struct {
	EntityID   ecs.EntityID // Entity checked
	ItemID     ecs.EntityID // Item checked
	IsEquipped bool         // Whether the item is equipped
	Slot       string       // Where the item is equipped, if applicable
	QueryID    string       // Matching ID from the request
}

// Type returns the event type
func (e EquipmentQueryResponseEvent) Type() ecs.EventType {
	return EventEquipmentResponse
}

// CameraUpdateEvent is emitted when the camera position changes
type CameraUpdateEvent struct {
	CameraID  ecs.EntityID // ID of the camera entity
	X         int          // New X position
	Y         int          // New Y position
	TargetID  ecs.EntityID // ID of the entity the camera is following (optional)
	ViewportW int          // Viewport width (optional)
	ViewportH int          // Viewport height (optional)
}

// Type returns the event type
func (e CameraUpdateEvent) Type() ecs.EventType {
	return EventCameraUpdate
}

// InventoryUIEvent is emitted for inventory UI interactions
type InventoryUIEvent struct {
	Action    string       // "open", "close", "select_item", "view_details", etc.
	EntityID  ecs.EntityID // Entity whose inventory is being interacted with
	ItemIndex int          // Index of the item being interacted with (if applicable)
	ItemID    ecs.EntityID // ID of the item being interacted with (if applicable)
}

// Type returns the event type
func (e InventoryUIEvent) Type() ecs.EventType {
	return EventInventoryUI
}

// ExamineEvent is emitted when an entity is examined
type ExamineEvent struct {
	TargetID ecs.EntityID // Entity being examined
}

// Type returns the event type
func (e ExamineEvent) Type() ecs.EventType {
	return EventExamine
}
