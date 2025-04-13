package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
)

// EquipmentSystem handles equipping, unequipping, and managing equipment effects
type EquipmentSystem struct {
	initialized bool
	world       *ecs.World
}

// NewEquipmentSystem creates a new equipment system
func NewEquipmentSystem() *EquipmentSystem {
	return &EquipmentSystem{
		initialized: false,
	}
}

// Update runs every frame to perform system logic
func (s *EquipmentSystem) Update(world *ecs.World, dt float64) {
	if !s.initialized {
		s.Initialize(world)
	}
}

// Initialize sets up the equipment system
func (s *EquipmentSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}
	s.world = world

	// Subscribe to equipment-related events
	world.GetEventManager().Subscribe(EventEquipItem, func(event ecs.Event) {
		req := event.(EquipItemRequestEvent)
		if req.SlotHint == "" {
			// Auto-equip
			err := s.EquipItemAuto(req.EntityID, req.ItemID)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Failed to equip item: %v", err))
			}
		} else {
			// Equip to specific slot
			var slot components.EquipmentSlot
			switch req.SlotHint {
			case "head":
				slot = components.SlotHead
			case "body":
				slot = components.SlotBody
			case "mainhand":
				slot = components.SlotMainHand
			case "offhand":
				slot = components.SlotOffHand
			case "feet":
				slot = components.SlotFeet
			case "accessory":
				slot = components.SlotAccessory
			default:
				GetMessageLog().Add(fmt.Sprintf("Unknown equipment slot: %s", req.SlotHint))
				return
			}

			err := s.EquipItem(req.EntityID, req.ItemID, slot)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Failed to equip item: %v", err))
			}
		}
	})

	world.GetEventManager().Subscribe(EventUnequipItem, func(event ecs.Event) {
		req := event.(UnequipItemRequestEvent)
		success := s.UnequipItemByID(req.EntityID, req.ItemID)
		if !success {
			GetMessageLog().Add("Failed to unequip item")
		}
	})

	world.GetEventManager().Subscribe(EventEquipmentQuery, func(event ecs.Event) {
		req := event.(EquipmentQueryRequestEvent)
		isEquipped := s.IsItemEquipped(req.EntityID, req.ItemID)

		slot := ""
		if isEquipped {
			// Determine which slot the item is in
			equipComp, exists := world.GetComponent(req.EntityID, components.Equipment)
			if exists {
				equipment := equipComp.(*components.EquipmentComponent)
				for slotName, itemID := range equipment.EquippedItems {
					if itemID == req.ItemID {
						slot = string(slotName)
						break
					}
				}
			}
		}

		// Send response using the proper event type
		world.EmitEvent(EquipmentQueryResponseEvent{
			EntityID:   req.EntityID,
			ItemID:     req.ItemID,
			IsEquipped: isEquipped,
			Slot:       slot,
			QueryID:    req.QueryID,
		})
	})

	s.initialized = true
}

// EquipItem equips an item to a slot
func (s *EquipmentSystem) EquipItem(entityID ecs.EntityID, itemID ecs.EntityID, slot components.EquipmentSlot) error {
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
func (s *EquipmentSystem) EquipItemAuto(entityID, itemID ecs.EntityID) error {
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

// IsItemEquipped checks if an item is already equipped by the entity
func (s *EquipmentSystem) IsItemEquipped(entityID, itemID ecs.EntityID) bool {
	// Get equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
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

// UnequipItemByID finds which slot an item is equipped in and unequips it
func (s *EquipmentSystem) UnequipItemByID(entityID, itemID ecs.EntityID) bool {
	// Get equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
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
			GetMessageLog().Add(fmt.Sprintf("Unequipped %s", s.getItemName(s.world, itemID)))
			return true
		}
	}

	return false
}

// UnequipItem removes an item from a slot and removes its effects
func (s *EquipmentSystem) UnequipItem(entityID ecs.EntityID, slot components.EquipmentSlot) error {
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

// RemoveAllEquipmentEffects removes all effects from all equipped items for an entity
func (s *EquipmentSystem) RemoveAllEquipmentEffects(entityID ecs.EntityID) error {
	// Get equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		return nil // No equipment component, nothing to do
	}
	equipment := equipComp.(*components.EquipmentComponent)

	// Go through all slots and remove effects
	for _, slot := range []components.EquipmentSlot{
		components.SlotHead, components.SlotBody, components.SlotMainHand,
		components.SlotOffHand, components.SlotFeet, components.SlotAccessory,
	} {
		itemID := equipment.GetEquippedItem(slot)
		if itemID != 0 {
			effects, exists := equipment.ActiveEffects[itemID]
			if exists {
				// Remove effects
				err := RemoveEntityEffects(s.world, entityID, effects)
				if err != nil {
					GetMessageLog().Add(fmt.Sprintf("Error removing effects from %s: %v",
						s.getItemName(s.world, itemID), err))
				}
				equipment.RemoveEffects(itemID)
			}
		}
	}

	return nil
}

// RemoveEquipmentEffects removes the effects of an equipped item from a specific slot
func (s *EquipmentSystem) RemoveEquipmentEffects(entityID ecs.EntityID, slot components.EquipmentSlot) bool {
	// Get equipment component
	equipComp, hasEquip := s.world.GetComponent(entityID, components.Equipment)
	if !hasEquip {
		return false
	}

	equipment := equipComp.(*components.EquipmentComponent)

	// Get the item in the slot
	itemID := equipment.GetEquippedItem(slot)
	if itemID == 0 {
		return false
	}

	// Get the item component
	itemComp, hasItem := s.world.GetComponent(itemID, components.Item)
	if !hasItem {
		return false
	}

	item := itemComp.(*components.ItemComponent)

	// Remove effects of the item
	if item.Data != nil {
		if effects, ok := item.Data.([]components.ItemEffect); ok {
			err := RemoveEntityEffects(s.world, entityID, effects)
			if err != nil {
				GetMessageLog().Add(fmt.Sprintf("Error removing effects: %v", err))
			}
		}
	}

	return true
}

// AddItemEffect adds an effect to an entity with the given parameters
func (s *EquipmentSystem) AddItemEffect(entityID ecs.EntityID, effect components.ItemEffect) error {
	// Use our utility function to apply the effect
	err := ApplyEntityEffect(s.world, entityID, effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		return fmt.Errorf("failed to apply effect: %v", err)
	}
	return nil
}

// RemoveItemEffect removes an effect from an entity
func (s *EquipmentSystem) RemoveItemEffect(entityID ecs.EntityID, effect components.ItemEffect) error {
	// Get the inverse effect
	inverseOp, inverseVal, err := CreateInverseEffect(effect.Component, effect.Property, effect.Operation, effect.Value)
	if err != nil {
		return fmt.Errorf("failed to create inverse effect: %v", err)
	}

	// Apply the inverse effect
	err = ApplyEntityEffect(s.world, entityID, effect.Component, effect.Property, inverseOp, inverseVal)
	if err != nil {
		return fmt.Errorf("failed to remove effect: %v", err)
	}

	return nil
}

// GetInverseEffect creates an inverse effect for removing effects
func (s *EquipmentSystem) GetInverseEffect(effect components.ItemEffect) (components.ItemEffect, error) {
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

// Helper function to get the name of an item
func (s *EquipmentSystem) getItemName(world *ecs.World, itemID ecs.EntityID) string {
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		return nameComp.(*components.NameComponent).Name
	}
	return "unknown item"
}
