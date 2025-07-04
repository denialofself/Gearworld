package components

import (
	"ebiten-rogue/ecs"
)

// Define component IDs for our game
const (
	Position ecs.ComponentID = iota
	Renderable
	Pl
	Collision
	AI
	MapComponentID
	Appearance // New component for custom tile appearances
	Camera     // Camera component for viewport management
	Player
	Stats
	MapType        // Map type component for distinguishing between world map and dungeons
	Name           // Name component for storing entity display names
	MapContext     // Map context component for tracking which map an entity belongs to
	Inventory      // Inventory component for storing items
	Item           // Item component for collectible objects
	FOV            // Field of vision component
	Equipment      // Equipment component for equipped items
	Container      // Container component for containers that can hold items
	Rotation       // Rotation component for storing entity rotation
	Effect         // Effect component for managing entity effects
	MonsterAbility // Monster ability component for special abilities
)
