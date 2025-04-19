package systems

import (
	"fmt"
	"sync"
	"time"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// InventorySystem handles inventory-related functionality
type InventorySystem struct {
	world                   *ecs.World
	pendingEquipmentQueries map[string]chan EquipmentQueryResponseEvent
	queryMutex              sync.Mutex
}

// NewInventorySystem creates a new inventory system
func NewInventorySystem() *InventorySystem {
	return &InventorySystem{
		pendingEquipmentQueries: make(map[string]chan EquipmentQueryResponseEvent),
	}
}

// Initialize sets up the inventory system
func (s *InventorySystem) Initialize(world *ecs.World) {
	s.world = world

	// Subscribe to equipment query responses
	world.GetEventManager().Subscribe(EventEquipmentResponse, func(event ecs.Event) {
		resp, ok := event.(EquipmentQueryResponseEvent)
		if !ok {
			return // Not a response event
		}

		s.queryMutex.Lock()
		defer s.queryMutex.Unlock()

		// Check if we have a pending query with this ID
		if ch, ok := s.pendingEquipmentQueries[resp.QueryID]; ok {
			// Send the response
			ch <- resp
			// Remove the pending query
			delete(s.pendingEquipmentQueries, resp.QueryID)
		}
	})
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

// checkItemEquipped sends a query event to check if an item is equipped and waits for the response
func (s *InventorySystem) checkItemEquipped(entityID, itemID ecs.EntityID) bool {
	// Generate a unique query ID
	queryID := fmt.Sprintf("eq_query_%d_%d_%d", entityID, itemID, time.Now().UnixNano())

	// Create a channel to receive the response
	respCh := make(chan EquipmentQueryResponseEvent, 1)

	// Register the query
	s.queryMutex.Lock()
	s.pendingEquipmentQueries[queryID] = respCh
	s.queryMutex.Unlock()

	// Send the query event
	s.world.EmitEvent(EquipmentQueryRequestEvent{
		EntityID: entityID,
		ItemID:   itemID,
		QueryID:  queryID,
	})

	// Wait for the response with a timeout
	select {
	case resp := <-respCh:
		return resp.IsEquipped
	case <-time.After(500 * time.Millisecond):
		// Timeout - clean up and return false
		s.queryMutex.Lock()
		delete(s.pendingEquipmentQueries, queryID)
		s.queryMutex.Unlock()
		GetMessageLog().Add("Equipment query timed out")
		return false
	}
}

// UseItem attempts to use the item at the given index in the player's inventory
func (s *InventorySystem) UseItem(world *ecs.World, playerID ecs.EntityID, itemIndex int) bool {
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

	// Get item component
	itemComp, exists := world.GetComponent(itemID, components.Item)
	if !exists {
		return false
	}
	item := itemComp.(*components.ItemComponent)

	// Check item type and handle accordingly
	if item.ItemType == "potion" || item.ItemType == "scroll" || item.ItemType == "food" || item.ItemType == "first aid" {
		// This is a consumable item
		if item.Data != nil {
			if _, ok := item.Data.([]components.GameEffect); ok {
				// Emit a single effects event for the item
				world.EmitEvent(EffectsEvent{
					EntityID:    playerID,
					EffectType:  "item",
					Property:    "",  // Not used when applying item effects
					Value:       nil, // Not used when applying item effects
					Source:      itemID,
					DisplayText: fmt.Sprintf("Used %s", s.getItemName(world, itemID)),
				})
			}
		}

		// Remove the item from inventory
		inventory.RemoveItem(itemID)
		GetMessageLog().Add(fmt.Sprintf("You used the %s.", s.getItemName(world, itemID)))
		return true
	} else if item.ItemType == "weapon" || item.ItemType == "armor" || item.ItemType == "headgear" ||
		item.ItemType == "shield" || item.ItemType == "ring" || item.ItemType == "amulet" {
		// This is an equippable item
		// Check if the item is already equipped using an event
		isEquipped := s.checkItemEquipped(playerID, itemID)

		if isEquipped {
			// If it's equipped, unequip it
			world.EmitEvent(UnequipItemRequestEvent{
				EntityID: playerID,
				ItemID:   itemID,
			})
			return true
		} else {
			// Otherwise, try to equip it
			world.EmitEvent(EquipItemRequestEvent{
				EntityID: playerID,
				ItemID:   itemID,
				SlotHint: "", // Auto-determine slot
			})
			return true
		}
	}

	GetMessageLog().Add(fmt.Sprintf("You can't use the %s.", s.getItemName(world, itemID)))
	return false
}

// HandleUseKeyPress handles the 'U' key for consuming items
func (s *InventorySystem) HandleUseKeyPress(world *ecs.World, playerID ecs.EntityID, selectedItemIndex int) bool {
	// Check if the index is valid
	if selectedItemIndex < 0 {
		GetMessageLog().Add("No item selected.")
		return false
	}

	// Use the item with our existing UseItem method
	return s.UseItem(world, playerID, selectedItemIndex)
}

// getItemName gets the name of an item
func (s *InventorySystem) getItemName(world *ecs.World, itemID ecs.EntityID) string {
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		return nameComp.(*components.NameComponent).Name
	}
	return "unknown item"
}

// IsItemConsumable checks if an item can be consumed/used directly
func (s *InventorySystem) IsItemConsumable(world *ecs.World, itemID ecs.EntityID) bool {
	// Get item component
	itemComp, exists := world.GetComponent(itemID, components.Item)
	if !exists {
		return false
	}
	item := itemComp.(*components.ItemComponent)

	// Check item type first
	if item.ItemType == "potion" || item.ItemType == "scroll" ||
		item.ItemType == "first aid" || item.ItemType == "bandage" {
		return true
	}

	// Check for consumable flag in data
	if data, ok := item.Data.(map[string]interface{}); ok {
		if consumable, ok := data["consumable"].(bool); ok && consumable {
			return true
		}

		// Check for consumable in tags
		if tags, ok := data["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok && tagStr == "consumable" {
					return true
				}
			}
		}
	}

	// Check if data is directly an array of GameEffect
	if effects, ok := item.Data.([]components.GameEffect); ok && len(effects) > 0 {
		// If item has effects and isn't equippable, it's likely consumable
		equippable := item.ItemType == "weapon" || item.ItemType == "armor" ||
			item.ItemType == "headgear" || item.ItemType == "shield" ||
			item.ItemType == "boots" || item.ItemType == "accessory"

		return !equippable
	}

	return false
}
