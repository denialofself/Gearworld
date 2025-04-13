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

	// Check if item is consumable
	isConsumable := false

	// Look for consumable flag in item data
	if data, ok := item.Data.(map[string]interface{}); ok {
		if consumable, ok := data["consumable"].(bool); ok && consumable {
			isConsumable = true
		}

		// Check for consumable in tags
		if tags, ok := data["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok && tagStr == "consumable" {
					isConsumable = true
					break
				}
			}
		}
	}

	// Handle based on item type or consumable status
	if isConsumable || item.ItemType == "potion" || item.ItemType == "scroll" ||
		item.ItemType == "first aid" || item.ItemType == "bandage" {
		// This is a consumable item - apply its effects
		itemName := s.getItemName(world, itemID)

		// Different use messages based on item type
		switch item.ItemType {
		case "potion":
			GetMessageLog().Add(fmt.Sprintf("You drink the %s.", itemName))
		case "scroll":
			GetMessageLog().Add(fmt.Sprintf("You read the %s.", itemName))
		default:
			GetMessageLog().Add(fmt.Sprintf("You use the %s.", itemName))
		}

		// Apply effects directly using our new system
		if item.Data != nil {
			if effects, ok := item.Data.([]components.ItemEffect); ok {
				// Apply all effects at once
				err := ApplyEntityEffects(world, playerID, effects)
				if err != nil {
					GetMessageLog().Add(fmt.Sprintf("Error applying effects: %v", err))
				} else {
					// Different effect messages based on primary effect
					hasPrimaryEffectMessage := false
					for _, effect := range effects {
						if effect.Component == "Stats" && effect.Property == "Health" {
							GetMessageLog().Add(fmt.Sprintf("The %s heals your wounds.", itemName))
							hasPrimaryEffectMessage = true
							break
						}
					}

					// Generic message if no specific effect message was shown
					if !hasPrimaryEffectMessage && len(effects) > 0 {
						GetMessageLog().Add(fmt.Sprintf("You feel the effects of the %s.", itemName))
					}
				}
			}
		}

		// Remove from inventory after use (it's consumable)
		inventory.RemoveItem(itemID)
		return true
	} else if item.ItemType == "weapon" || item.ItemType == "armor" || item.ItemType == "headgear" ||
		item.ItemType == "shield" || item.ItemType == "ring" || item.ItemType == "amulet" {
		// This is an equippable item
		// Check if the item is already equipped
		if s.isItemEquipped(world, playerID, itemID) {
			// If it's equipped, unequip it
			if s.unequipItemByID(world, playerID, itemID) {
				return true
			}
		} else {
			// Otherwise, try to equip it
			if s.directEquipItem(world, playerID, itemID) {
				return true
			}
		}
	}

	GetMessageLog().Add(fmt.Sprintf("You can't use the %s.", s.getItemName(world, itemID)))
	return false
}

// directEquipItem is a simplified equipment function that uses the effects system
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
	case "headgear":
		slot = components.SlotHead
	case "boots":
		slot = components.SlotFeet
	case "accessory":
		slot = components.SlotAccessory
	default:
		GetMessageLog().Add(fmt.Sprintf("Cannot equip: unknown item type '%s'", item.ItemType))
		return false
	}

	// Use the proper EquipItem method that handles all the details
	err := s.EquipItem(entityID, itemID, slot)
	if err != nil {
		GetMessageLog().Add(fmt.Sprintf("Cannot equip: %v", err))
		return false
	}

	return true
}

// EquipItem equips an item to a slot
func (s *InventorySystem) EquipItem(entityID ecs.EntityID, itemID ecs.EntityID, slot components.EquipmentSlot) error {
	// Get the item component
	itemComp, exists := s.world.GetComponent(itemID, components.Item)
	if !exists {
		return fmt.Errorf("entity is not an item")
	}
	item := itemComp.(*components.ItemComponent)

	// Get or create equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		equipComp = components.NewEquipmentComponent()
		s.world.AddComponent(entityID, components.Equipment, equipComp)
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Unequip any existing item in the slot
	if oldItemID := equipment.GetEquippedItem(slot); oldItemID != 0 {
		s.UnequipItem(entityID, slot)
		GetMessageLog().Add(fmt.Sprintf("Unequipped previous item from %s slot", slot))
	}

	// Equip the new item
	equipment.EquipItem(slot, itemID)

	// Process the item effects
	if item.Data != nil {
		if effects, ok := item.Data.([]components.ItemEffect); ok {
			GetMessageLog().Add(fmt.Sprintf("Applying %d effects from %s", len(effects), s.getItemName(s.world, itemID)))

			// Track the effects in the equipment component
			for _, effect := range effects {
				equipment.AddEffect(itemID, effect)
			}

			// Apply all effects at once using our utility
			err := ApplyEntityEffects(s.world, entityID, effects)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Error applying effects: %v", err))
			}
		} else {
			GetMessageLog().Add(fmt.Sprintf("Item data is not []ItemEffect but %T", item.Data))
		}
	} else {
		GetMessageLog().Add(fmt.Sprintf("Item %s has no effects data", s.getItemName(s.world, itemID)))
	}

	// Get item name for message
	itemName := s.getItemName(s.world, itemID)

	// Emit an equipment event that other systems might be interested in
	s.world.EmitEvent(ItemEquippedEvent{
		EntityID: entityID,
		ItemID:   itemID,
		Slot:     string(slot),
	})

	GetMessageLog().Add(fmt.Sprintf("Equipped %s to %s slot", itemName, slot))
	return nil
}

// EquipItemAuto equips an item to the appropriate slot based on its type
func (s *InventorySystem) EquipItemAuto(entityID, itemID ecs.EntityID) error {
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
	case "headgear":
		slot = components.SlotHead
	case "boots":
		slot = components.SlotFeet
	case "accessory":
		slot = components.SlotAccessory
	default:
		return fmt.Errorf("item has unknown type: %s", item.ItemType)
	}

	// Equip to the determined slot
	return s.EquipItem(entityID, itemID, slot)
}

// isItemEquipped checks if an item is already equipped by the entity
func (s *InventorySystem) isItemEquipped(world *ecs.World, entityID, itemID ecs.EntityID) bool {
	// Get equipment component
	equipComp, exists := world.GetComponent(entityID, components.Equipment)
	if !exists {
		return false
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Check all equipment slots to see if the item is equipped in any of them
	for _, slot := range []components.EquipmentSlot{
		components.SlotHead, components.SlotBody, components.SlotMainHand,
		components.SlotOffHand, components.SlotFeet, components.SlotAccessory,
	} {
		if equipment.GetEquippedItem(slot) == itemID {
			return true
		}
	}

	return false
}

// unequipItemByID finds which slot an item is equipped in and unequips it
func (s *InventorySystem) unequipItemByID(world *ecs.World, entityID, itemID ecs.EntityID) bool {
	// Get equipment component
	equipComp, exists := world.GetComponent(entityID, components.Equipment)
	if !exists {
		return false
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Check all equipment slots to find the item
	for _, slot := range []components.EquipmentSlot{
		components.SlotHead, components.SlotBody, components.SlotMainHand,
		components.SlotOffHand, components.SlotFeet, components.SlotAccessory,
	} {
		if equipment.GetEquippedItem(slot) == itemID {
			// Found the item, unequip it
			err := s.UnequipItem(entityID, slot)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Failed to unequip: %v", err))
				return false
			}
			GetMessageLog().Add(fmt.Sprintf("Unequipped %s", s.getItemName(world, itemID)))
			return true
		}
	}

	return false
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

	// Get the item's effects
	effects, exists := equipment.ActiveEffects[itemID]
	if exists {
		GetMessageLog().Add(fmt.Sprintf("Removing %d effects from %s", len(effects), s.getItemName(s.world, itemID)))

		// Remove all effects at once using our utility
		err := RemoveEntityEffects(s.world, entityID, effects)
		if err != nil {
			GetMessageLog().Add(fmt.Sprintf("Error removing effects: %v", err))
		}
	}

	// Remove the tracked effects from the equipment component
	equipment.RemoveEffects(itemID)

	// Unequip the item
	equipment.UnequipItem(slot)

	// Emit an unequip event
	s.world.EmitEvent(ItemUnequippedEvent{
		EntityID: entityID,
		ItemID:   itemID,
		Slot:     string(slot),
	})

	// Log the unequipment
	GetMessageLog().Add(fmt.Sprintf("Unequipped %s", s.getItemName(s.world, itemID)))
	return nil
}

// getItemName gets the name of an item
func (s *InventorySystem) getItemName(world *ecs.World, itemID ecs.EntityID) string {
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		return nameComp.(*components.NameComponent).Name
	}
	return "unknown item"
}

// Apply item effect based on JSON definition - generic implementation
func (s *InventorySystem) ApplyItemEffect(world *ecs.World, entityID, itemID ecs.EntityID, effect components.ItemEffect) {
	// Log debug info to track effect application
	GetMessageLog().Add(fmt.Sprintf("Applying %s.%s effect (%s %v)",
		effect.Component, effect.Property, effect.Operation, effect.Value))

	// Apply the effect using our utility function
	err := ApplyEntityEffect(world, entityID, effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		GetMessageLog().Add(fmt.Sprintf("Error applying effect: %v", err))
		return
	}

	// Log successful application
	GetMessageLog().Add(fmt.Sprintf("%s.%s %s to %v applied successfully",
		effect.Component, effect.Property, effect.Operation, effect.Value))
}

// Remove item effect based on JSON definition - generic implementation
func (s *InventorySystem) RemoveItemEffect(world *ecs.World, entityID, itemID ecs.EntityID, effect components.ItemEffect) {
	// Log debug info to track effect removal
	GetMessageLog().Add(fmt.Sprintf("Removing %s.%s effect (%s %v)",
		effect.Component, effect.Property, effect.Operation, effect.Value))

	// Create inverse effect
	inverseOp, inverseVal, err := CreateInverseEffect(
		effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		GetMessageLog().Add(fmt.Sprintf("Error creating inverse effect: %v", err))
		return
	}

	// Apply the inverse effect
	err = ApplyEntityEffect(world, entityID, effect.Component, effect.Property, inverseOp, inverseVal)
	if err != nil {
		GetMessageLog().Add(fmt.Sprintf("Error applying inverse effect: %v", err))
		return
	}

	// Log successful removal
	GetMessageLog().Add(fmt.Sprintf("Successfully removed effect from %s.%s",
		effect.Component, effect.Property))
}

// createInverseEffect creates the inverse of an effect for removal (legacy function, kept for reference)
func createInverseEffect(effect components.ItemEffect) components.ItemEffect {
	inverseEffect := effect // Copy the original effect

	// Use the utility function to create inverse
	inverseOp, inverseVal, err := CreateInverseEffect(
		effect.Component, effect.Property, effect.Operation, effect.Value)
	if err == nil {
		inverseEffect.Operation = inverseOp
		inverseEffect.Value = inverseVal
	} else {
		// Fallback to old behavior if there's an error
		switch effect.Operation {
		case "add":
			// For numeric values, negate them
			switch v := effect.Value.(type) {
			case int:
				inverseEffect.Value = -v
			case float64:
				inverseEffect.Value = -v
			}
		case "set":
			// For "set" operations, use defaults based on property
			switch effect.Component {
			case "Stats":
				switch effect.Property {
				case "Attack":
					inverseEffect.Value = 1 // Default attack
				case "Defense":
					inverseEffect.Value = 0 // Default defense
				}
			case "FOV":
				switch effect.Property {
				case "Range":
					inverseEffect.Value = 8 // Default FOV range
				case "LightRange":
					inverseEffect.Value = 0 // Default light range
				case "LightSource":
					inverseEffect.Value = false // Default light source state
				}
			}
		}
	}

	return inverseEffect
}

// Note: These methods are no longer needed as their functionality is now handled
// in the UnequipItem method which uses the effects system to cancel effects
// They're kept here as comments for reference but should be removed in future refactoring

/*
// removeItemEffects removes all effects from an item
func (s *InventorySystem) removeItemEffects(entityID, itemID ecs.EntityID) {
    // This functionality has been moved to UnequipItem which uses the effects system
}

// removeItemEffect removes a single effect
func (s *InventorySystem) removeItemEffect(entityID ecs.EntityID, effect components.ItemEffect) {
    // This functionality has been moved to UnequipItem which uses the effects system
}
*/

// HandleUseKeyPress handles when the player presses the 'U' key to use the currently selected item
func (s *InventorySystem) HandleUseKeyPress(world *ecs.World, playerID ecs.EntityID, selectedItemIndex int) bool {
	// Check if the index is valid
	if selectedItemIndex < 0 {
		GetMessageLog().Add("No item selected.")
		return false
	}

	// Use the item with our existing UseItem method
	return s.UseItem(world, playerID, selectedItemIndex)
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

	// Check if data is directly an array of ItemEffect
	if effects, ok := item.Data.([]components.ItemEffect); ok && len(effects) > 0 {
		// If item has effects and isn't equippable, it's likely consumable
		equippable := item.ItemType == "weapon" || item.ItemType == "armor" ||
			item.ItemType == "headgear" || item.ItemType == "shield" ||
			item.ItemType == "boots" || item.ItemType == "accessory"

		return !equippable
	}

	return false
}

// RemoveEquippedItemEffects removes the effects of an equipped item
func (s *InventorySystem) RemoveEquippedItemEffects(world *ecs.World, entityID ecs.EntityID, slot string) bool {
	// Get equipment component
	equipComp, hasEquip := world.GetComponent(entityID, components.Equipment)
	if !hasEquip {
		return false
	}

	equipment := equipComp.(*components.EquipmentComponent)

	// Convert string slot to EquipmentSlot
	var equipSlot components.EquipmentSlot
	switch slot {
	case "mainhand":
		equipSlot = components.SlotMainHand
	case "offhand":
		equipSlot = components.SlotOffHand
	case "head":
		equipSlot = components.SlotHead
	case "body":
		equipSlot = components.SlotBody
	case "feet":
		equipSlot = components.SlotFeet
	case "accessory":
		equipSlot = components.SlotAccessory
	default:
		return false
	}

	// Get the item in the slot
	itemID := equipment.GetEquippedItem(equipSlot)
	if itemID == 0 {
		return false
	}

	// Get the item component
	itemComp, hasItem := world.GetComponent(itemID, components.Item)
	if !hasItem {
		return false
	}

	item := itemComp.(*components.ItemComponent)

	// Remove effects of the item
	if item.Data != nil {
		if effects, ok := item.Data.([]components.ItemEffect); ok {
			err := RemoveEntityEffects(world, entityID, effects)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Error removing effects: %v", err))
			}
		}
	}

	return true
}

// AddItemEffect adds an effect to an entity with the given parameters
func (s *InventorySystem) AddItemEffect(world *ecs.World, entityID ecs.EntityID, effect components.ItemEffect) error {
	// Use our utility function to apply the effect
	err := ApplyEntityEffect(world, entityID, effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		return fmt.Errorf("failed to apply effect: %v", err)
	}
	return nil
}

// GetInverseEffect creates an inverse effect for removing effects
func (s *InventorySystem) GetInverseEffect(effect components.ItemEffect) (components.ItemEffect, error) {
	inverseOp, inverseVal, err := CreateInverseEffect(effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		return components.ItemEffect{}, err
	}

	return components.ItemEffect{
		Component: effect.Component,
		Property:  effect.Property,
		Operation: inverseOp,
		Value:     inverseVal,
	}, nil
}
