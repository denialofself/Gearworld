package systems

import (
	"fmt"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// InventorySystem handles inventory-related functionality
type InventorySystem struct {
	world *ecs.World
}

// NewInventorySystem creates a new inventory system
func NewInventorySystem() *InventorySystem {
	return &InventorySystem{}
}

// Initialize sets up the inventory system
func (s *InventorySystem) Initialize(world *ecs.World) {
	s.world = world
}

// Update checks for item pickups and inventory interactions
func (s *InventorySystem) Update(world *ecs.World, dt float64) {
	s.world = world

	// Get the player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}
	playerEntity := playerEntities[0]

	// Check if player has inventory
	if !world.HasComponent(playerEntity.ID, components.Inventory) {
		return
	}

	// Get player position
	posComp, exists := world.GetComponent(playerEntity.ID, components.Position)
	if !exists {
		return
	}
	playerPos := posComp.(*components.PositionComponent)

	// Check if player is standing on any items
	s.checkItemPickups(world, playerEntity.ID, playerPos)
}

// checkItemPickups checks if the player is standing on any items and picks them up
func (s *InventorySystem) checkItemPickups(world *ecs.World, playerID ecs.EntityID, playerPos *components.PositionComponent) {
	// Get all items
	itemEntities := world.GetEntitiesWithTag("item")

	// Get player's inventory
	invComp, exists := world.GetComponent(playerID, components.Inventory)
	if !exists {
		return
	}
	inventory := invComp.(*components.InventoryComponent)

	// Check each item to see if it's at the player's position
	for _, itemEntity := range itemEntities {
		// Skip items that are already in an inventory
		if !world.HasComponent(itemEntity.ID, components.Position) {
			continue
		}

		itemPosComp, exists := world.GetComponent(itemEntity.ID, components.Position)
		if !exists {
			continue
		}
		itemPos := itemPosComp.(*components.PositionComponent)

		// If item is at player's position, pick it up
		if itemPos.X == playerPos.X && itemPos.Y == playerPos.Y {
			s.pickupItem(world, playerID, itemEntity.ID, inventory)
		}
	}
}

// pickupItem adds an item to the player's inventory and removes it from the map
func (s *InventorySystem) pickupItem(world *ecs.World, playerID ecs.EntityID, itemID ecs.EntityID, inventory *components.InventoryComponent) {
	// Check if inventory has space
	if inventory.IsFull() {
		GetMessageLog().Add("Your inventory is full.")
		return
	}

	// Get item name for message
	var itemName string = "an item"
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		itemName = nameComp.(*components.NameComponent).Name
	}

	// Add item to inventory
	success := inventory.AddItem(itemID)
	if success {
		// Remove position component from the item (it's now in inventory)
		world.RemoveComponent(itemID, components.Position)

		// Log the pickup
		GetMessageLog().Add(fmt.Sprintf("You picked up %s.", itemName))
	}
}

// DropItem drops an item from inventory to the map
func (s *InventorySystem) DropItem(world *ecs.World, playerID ecs.EntityID, itemIndex int) bool {
	// Get player inventory
	invComp, exists := world.GetComponent(playerID, components.Inventory)
	if !exists {
		return false
	}
	inventory := invComp.(*components.InventoryComponent)

	// Check if index is valid
	if itemIndex < 0 || itemIndex >= inventory.Size() {
		return false
	}

	// Get item ID
	itemID := inventory.GetItemByIndex(itemIndex)
	if itemID == 0 {
		return false
	}

	// Get player position
	posComp, exists := world.GetComponent(playerID, components.Position)
	if !exists {
		return false
	}
	playerPos := posComp.(*components.PositionComponent)

	// Get item name
	var itemName string = "an item"
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		itemName = nameComp.(*components.NameComponent).Name
	}

	// Add position component to the item (it's now on the map)
	world.AddComponent(itemID, components.Position, &components.PositionComponent{
		X: playerPos.X,
		Y: playerPos.Y,
	})

	// Remove from inventory
	inventory.RemoveItem(itemID)

	// Log the drop
	GetMessageLog().Add(fmt.Sprintf("You dropped %s.", itemName))

	return true
}
