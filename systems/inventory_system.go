package systems

import (
	"fmt"
	"reflect"

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

// UseItem uses the item at the given index in the player's inventory
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
	// Handle based on item type
	switch item.ItemType {
	case "potion":
		// Implement potion effects
		GetMessageLog().Add(fmt.Sprintf("You drink the %s.", s.getItemName(world, itemID)))
		// Remove from inventory after use
		inventory.RemoveItem(itemID)
		return true

	case "scroll":
		// Implement scroll effects
		GetMessageLog().Add(fmt.Sprintf("You read the %s.", s.getItemName(world, itemID)))
		// Remove from inventory after use
		inventory.RemoveItem(itemID)
		return true
	case "weapon", "armor", "helmet", "shield", "boots", "accessory":
		// Simple direct equipping of the item
		if s.directEquipItem(world, playerID, itemID) {
			return true
		}
	}

	GetMessageLog().Add(fmt.Sprintf("You can't use the %s.", s.getItemName(world, itemID)))
	return false
}

// directEquipItem is a simplified equipment function that directly manages the equipment component
func (s *InventorySystem) directEquipItem(world *ecs.World, entityID, itemID ecs.EntityID) bool {
	// Get the item component
	itemComp, exists := world.GetComponent(itemID, components.Item)
	if !exists {
		GetMessageLog().Add("Cannot equip: missing item component")
		return false
	}
	item := itemComp.(*components.ItemComponent)

	// Determine the appropriate slot based on item type
	var slot components.EquipmentSlot
	switch item.ItemType {
	case "weapon":
		slot = components.SlotMainHand
	case "armor":
		slot = components.SlotBody
	case "shield":
		slot = components.SlotOffHand
	case "helmet":
		slot = components.SlotHead
	case "boots":
		slot = components.SlotFeet
	case "accessory":
		slot = components.SlotAccessory
	default:
		GetMessageLog().Add(fmt.Sprintf("Cannot equip: unknown item type '%s'", item.ItemType))
		return false
	}

	// Get or create equipment component
	var equipComp *components.EquipmentComponent
	if comp, exists := world.GetComponent(entityID, components.Equipment); exists {
		equipComp = comp.(*components.EquipmentComponent)
	} else {
		equipComp = components.NewEquipmentComponent()
		world.AddComponent(entityID, components.Equipment, equipComp)
	}

	// Unequip any existing item in the slot
	if oldItemID := equipComp.GetEquippedItem(slot); oldItemID != 0 {
		// Just remove it from the slot without complex effect handling
		equipComp.UnequipItem(slot)
		GetMessageLog().Add(fmt.Sprintf("Unequipped previous item from %s slot", slot))
	}

	// Simply equip the new item to the slot
	equipComp.EquipItem(slot, itemID)

	// Apply any item effects
	if item.Data != nil {
		if effects, ok := item.Data.([]components.ItemEffect); ok {
			for _, effect := range effects {
				equipComp.AddEffect(itemID, effect)
				// Apply the effect directly here
				s.applySimpleEffect(world, entityID, effect)
			}
		}
	}

	// Get item name for message
	itemName := "an item"
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		itemName = nameComp.(*components.NameComponent).Name
	}

	GetMessageLog().Add(fmt.Sprintf("Equipped %s to %s slot", itemName, slot))
	return true
}

// applySimpleEffect applies a simple effect to the appropriate component
func (s *InventorySystem) applySimpleEffect(world *ecs.World, entityID ecs.EntityID, effect components.ItemEffect) {
	switch effect.Component {
	case "FOV":
		// Handle FOV effects
		if fovComp, exists := world.GetComponent(entityID, components.FOV); exists {
			fov := fovComp.(*components.FOVComponent)

			switch effect.Property {
			case "Range":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						fov.Range += int(value)
					} else if value, ok := effect.Value.(int); ok {
						fov.Range += value
					}
				} else if effect.Operation == "set" {
					if value, ok := effect.Value.(float64); ok {
						fov.Range = int(value)
					} else if value, ok := effect.Value.(int); ok {
						fov.Range = value
					}
				}
			case "LightSource":
				if effect.Operation == "set" {
					if value, ok := effect.Value.(bool); ok {
						fov.LightSource = value
					}
				}
			case "LightRange":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						fov.LightRange += int(value)
					} else if value, ok := effect.Value.(int); ok {
						fov.LightRange += value
					}
				} else if effect.Operation == "set" {
					if value, ok := effect.Value.(float64); ok {
						fov.LightRange = int(value)
					} else if value, ok := effect.Value.(int); ok {
						fov.LightRange = value
					}
				}
			}
		}

	case "Stats":
		// Handle Stats effects - simplified approach
		if statsComp, exists := world.GetComponent(entityID, components.Stats); exists {
			stats := statsComp.(*components.StatsComponent)

			// Directly modify the stats based on the property
			switch effect.Property {
			case "Health":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						stats.Health += int(value)
					} else if value, ok := effect.Value.(int); ok {
						stats.Health += value
					}
				}
			case "MaxHealth":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						stats.MaxHealth += int(value)
					} else if value, ok := effect.Value.(int); ok {
						stats.MaxHealth += value
					}
				}
			case "Attack":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						stats.Attack += int(value)
					} else if value, ok := effect.Value.(int); ok {
						stats.Attack += value
					}
				}
			case "Defense":
				if effect.Operation == "add" {
					if value, ok := effect.Value.(float64); ok {
						stats.Defense += int(value)
					} else if value, ok := effect.Value.(int); ok {
						stats.Defense += value
					}
				}
			}
		}
	}
}

// getItemName gets the name of an item
func (s *InventorySystem) getItemName(world *ecs.World, itemID ecs.EntityID) string {
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		return nameComp.(*components.NameComponent).Name
	}
	return "unknown item"
}

// EquipItem equips an item to an entity and applies its effects
func (s *InventorySystem) EquipItem(entityID, itemID ecs.EntityID) error {
	// Get the item component
	itemComp, exists := s.world.GetComponent(itemID, components.Item)
	if !exists {
		return fmt.Errorf("item doesn't have Item component")
	}
	item := itemComp.(*components.ItemComponent)

	// Determine the appropriate slot based on item type
	var slot components.EquipmentSlot
	switch item.ItemType {
	case "weapon":
		slot = components.SlotMainHand
	case "armor":
		slot = components.SlotBody
	case "shield":
		slot = components.SlotOffHand
	case "helmet":
		slot = components.SlotHead
	case "boots":
		slot = components.SlotFeet
	case "accessory":
		slot = components.SlotAccessory
	default:
		return fmt.Errorf("item has unknown type: %s", item.ItemType)
	}

	// Get or create equipment component
	var equipComp *components.EquipmentComponent
	if comp, exists := s.world.GetComponent(entityID, components.Equipment); exists {
		equipComp = comp.(*components.EquipmentComponent)
	} else {
		equipComp = components.NewEquipmentComponent()
		s.world.AddComponent(entityID, components.Equipment, equipComp)
	}

	// Unequip any existing item in the slot
	if oldItemID := equipComp.GetEquippedItem(slot); oldItemID != 0 {
		s.UnequipItem(entityID, slot)
	}

	// Equip the new item
	equipComp.EquipItem(slot, itemID)

	// Extract effects from item data and apply them
	if item.Data != nil {
		if effects, ok := item.Data.([]components.ItemEffect); ok {
			for _, effect := range effects {
				equipComp.AddEffect(itemID, effect)
				s.applyItemEffect(entityID, itemID, effect)
			}
		}
	}

	// Log the equipment
	GetMessageLog().Add(fmt.Sprintf("Equipped %s", s.getItemName(s.world, itemID)))
	return nil
}

// UnequipItem removes an item from a slot and removes its effects
func (s *InventorySystem) UnequipItem(entityID ecs.EntityID, slot components.EquipmentSlot) error {
	// Get equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		return fmt.Errorf("entity doesn't have Equipment component")
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Get the item that's currently equipped
	itemID := equipment.GetEquippedItem(slot)
	if itemID == 0 {
		return fmt.Errorf("no item equipped in slot %s", slot)
	}

	// Remove the item's effects
	s.removeItemEffects(entityID, itemID)
	equipment.RemoveEffects(itemID)

	// Unequip the item
	equipment.UnequipItem(slot)

	// Log the unequipment
	GetMessageLog().Add(fmt.Sprintf("Unequipped %s", s.getItemName(s.world, itemID)))
	return nil
}

// applyItemEffect applies an effect to an entity
func (s *InventorySystem) applyItemEffect(entityID, itemID ecs.EntityID, effect components.ItemEffect) {
	switch effect.Component {
	case "FOV":
		s.applyFOVEffect(entityID, effect)
	case "Stats":
		s.applyStatsEffect(entityID, effect)
	}
}

// applyFOVEffect applies FOV-related effects
func (s *InventorySystem) applyFOVEffect(entityID ecs.EntityID, effect components.ItemEffect) {
	// Get FOV component
	fovComp, exists := s.world.GetComponent(entityID, components.FOV)
	if !exists {
		return
	}
	fov := fovComp.(*components.FOVComponent)

	// Apply the effect based on property and operation
	switch effect.Property {
	case "Range":
		if effect.Operation == "add" {
			if value, ok := effect.Value.(float64); ok {
				fov.Range += int(value)
			} else if value, ok := effect.Value.(int); ok {
				fov.Range += value
			}
		} else if effect.Operation == "set" {
			if value, ok := effect.Value.(float64); ok {
				fov.Range = int(value)
			} else if value, ok := effect.Value.(int); ok {
				fov.Range = value
			}
		}
	case "LightSource":
		if effect.Operation == "set" {
			if value, ok := effect.Value.(bool); ok {
				fov.LightSource = value
			}
		}
	case "LightRange":
		if effect.Operation == "add" {
			if value, ok := effect.Value.(float64); ok {
				fov.LightRange += int(value)
			} else if value, ok := effect.Value.(int); ok {
				fov.LightRange += value
			}
		} else if effect.Operation == "set" {
			if value, ok := effect.Value.(float64); ok {
				fov.LightRange = int(value)
			} else if value, ok := effect.Value.(int); ok {
				fov.LightRange = value
			}
		}
	}
}

// applyStatsEffect applies stat-related effects
func (s *InventorySystem) applyStatsEffect(entityID ecs.EntityID, effect components.ItemEffect) {
	// Get Stats component
	statsComp, exists := s.world.GetComponent(entityID, components.Stats)
	if !exists {
		return
	}
	stats := statsComp.(*components.StatsComponent)

	// Use reflection to get the field value
	statsValue := reflect.ValueOf(stats).Elem()
	field := statsValue.FieldByName(effect.Property)

	// Make sure the field exists and is an int
	if !field.IsValid() || field.Kind() != reflect.Int {
		return
	}

	// Apply the effect based on operation
	currentValue := int(field.Int())
	var newValue int

	switch effect.Operation {
	case "add":
		if value, ok := effect.Value.(float64); ok {
			newValue = currentValue + int(value)
		} else if value, ok := effect.Value.(int); ok {
			newValue = currentValue + value
		}
	case "multiply":
		if value, ok := effect.Value.(float64); ok {
			newValue = int(float64(currentValue) * value)
		} else if value, ok := effect.Value.(int); ok {
			newValue = currentValue * value
		}
	case "set":
		if value, ok := effect.Value.(float64); ok {
			newValue = int(value)
		} else if value, ok := effect.Value.(int); ok {
			newValue = value
		}
	}

	// Set the new value
	field.SetInt(int64(newValue))
}

// removeItemEffects removes all effects from an item
func (s *InventorySystem) removeItemEffects(entityID, itemID ecs.EntityID) {
	// Get equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		return
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Get effects for the item
	effects, exists := equipment.ActiveEffects[itemID]
	if !exists || len(effects) == 0 {
		return
	}

	// Remove each effect
	for _, effect := range effects {
		s.removeItemEffect(entityID, effect)
	}
}

// removeItemEffect removes a single effect
func (s *InventorySystem) removeItemEffect(entityID ecs.EntityID, effect components.ItemEffect) {
	// This is a simplified implementation that just reverses additions
	if effect.Operation == "add" {
		// Create a negative version of the effect
		inverseEffect := effect
		if value, ok := effect.Value.(float64); ok {
			inverseEffect.Value = -value
		} else if value, ok := effect.Value.(int); ok {
			inverseEffect.Value = -value
		}
		s.applyItemEffect(entityID, 0, inverseEffect)
	}
	// For "set" operations, you'd need to track the original value
}
