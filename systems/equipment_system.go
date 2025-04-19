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
	entity := s.world.GetEntity(entityID)
	if entity == nil {
		return fmt.Errorf("entity not found")
	}

	// Get the item component from the item entity
	itemComp, exists := s.world.GetComponent(itemID, components.Item)
	if !exists {
		return fmt.Errorf("item doesn't have Item component")
	}
	item, ok := itemComp.(*components.ItemComponent)
	if !ok {
		return fmt.Errorf("item component is not of type *ItemComponent")
	}

	// Get the equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		return fmt.Errorf("entity doesn't have Equipment component")
	}
	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return fmt.Errorf("invalid Equipment component type")
	}

	// Get the stats component
	statsComp, exists := s.world.GetComponent(entityID, components.Stats)
	if !exists {
		return fmt.Errorf("entity lacks Stats component")
	}
	stats, ok := statsComp.(*components.StatsComponent)
	if !ok {
		return fmt.Errorf("invalid Stats component type")
	}

	// Log the equip event and effects
	GetDebugLog().Add(fmt.Sprintf("Equipping item %d in slot %s", itemID, slot))
	if item.Data != nil {
		if effects, ok := item.Data.([]components.GameEffect); ok {
			GetDebugLog().Add(fmt.Sprintf("Item has %d effects:", len(effects)))
			for _, effect := range effects {
				GetDebugLog().Add(fmt.Sprintf("  - Effect: %s %s %v on %s.%s",
					effect.Type, effect.Operation, effect.Value,
					effect.Target.Component, effect.Target.Property))
			}
		}
	}

	// Log current stats before equip
	GetDebugLog().Add(fmt.Sprintf("Current stats before equip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Unequip any existing item in the slot
	if oldItemID := equipment.GetEquippedItem(slot); oldItemID != 0 {
		s.UnequipItem(entityID, slot)
		GetMessageLog().Add(fmt.Sprintf("Unequipped previous item from %s slot", slot))
	}

	// Equip the new item
	equipment.EquipItem(slot, itemID)

	// Process the item effects
	if item.Data != nil {
		if _, ok := item.Data.([]components.GameEffect); ok {
			// Emit event for effects to be applied
			s.world.EmitEvent(ItemEquippedEvent{
				EntityID: entityID,
				ItemID:   itemID,
				Slot:     string(slot),
			})
		} else {
			GetMessageLog().Add(fmt.Sprintf("Item data is not []GameEffect but %T", item.Data))
		}
	} else {
		GetMessageLog().Add(fmt.Sprintf("Item %s has no effects data", s.getItemName(s.world, itemID)))
	}

	// Get item name for message
	itemName := s.getItemName(s.world, itemID)

	// Log stats after equip
	GetDebugLog().Add(fmt.Sprintf("Stats after equip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

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
	item, ok := itemComp.(*components.ItemComponent)
	if !ok {
		return fmt.Errorf("item component is not of type *ItemComponent")
	}

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
	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return false
	}

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
	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return false
	}

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
	entity := s.world.GetEntity(entityID)
	if entity == nil {
		return fmt.Errorf("entity not found")
	}

	// Get the equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		return fmt.Errorf("entity doesn't have Equipment component")
	}
	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return fmt.Errorf("invalid Equipment component type")
	}

	// Get the stats component
	statsComp, exists := s.world.GetComponent(entityID, components.Stats)
	if !exists {
		return fmt.Errorf("entity lacks Stats component")
	}
	stats, ok := statsComp.(*components.StatsComponent)
	if !ok {
		return fmt.Errorf("invalid Stats component type")
	}

	// Log current stats before unequip
	GetDebugLog().Add(fmt.Sprintf("Unequipping item from slot %s", slot))
	GetDebugLog().Add(fmt.Sprintf("Current stats before unequip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Get the item that's currently equipped
	itemID := equipment.GetEquippedItem(slot)
	if itemID == 0 {
		return fmt.Errorf("no item equipped in slot %s", slot)
	}

	// Get the item component to access its effects
	itemComp, exists := s.world.GetComponent(entityID, components.Item)
	if !exists {
		return fmt.Errorf("equipped item lacks Item component")
	}
	item := itemComp.(components.ItemComponent)

	// Remove effects if the item has any
	if item.Data != nil {
		if effects, ok := item.Data.([]components.GameEffect); ok {
			GetDebugLog().Add(fmt.Sprintf("Removing %d effects from %s", len(effects), s.getItemName(s.world, itemID)))

			// Emit event for effects to be removed
			s.world.EmitEvent(ItemUnequippedEvent{
				EntityID: entityID,
				ItemID:   itemID,
				Slot:     string(slot),
			})
		}
	}

	// Remove the tracked effects from the equipment component
	equipment.RemoveEffects(itemID)

	// Unequip the item
	equipment.UnequipItem(slot)

	// Log stats after unequip
	GetDebugLog().Add(fmt.Sprintf("Stats after unequip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

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
	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return fmt.Errorf("invalid Equipment component type")
	}

	// Go through all slots and remove effects
	for _, slot := range []components.EquipmentSlot{
		components.SlotHead, components.SlotBody, components.SlotMainHand,
		components.SlotOffHand, components.SlotFeet, components.SlotAccessory,
	} {
		itemID := equipment.GetEquippedItem(slot)
		if itemID != 0 {
			if _, exists := equipment.ActiveEffects[itemID]; exists {
				// Emit event for effects to be removed
				s.world.EmitEvent(ItemUnequippedEvent{
					EntityID: entityID,
					ItemID:   itemID,
					Slot:     string(slot),
				})
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

	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return false
	}

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
		if _, ok := item.Data.([]components.GameEffect); ok {
			// Emit event for effects to be removed
			s.world.EmitEvent(ItemUnequippedEvent{
				EntityID: entityID,
				ItemID:   itemID,
				Slot:     string(slot),
			})
		}
	}

	return true
}

// AddItemEffect adds an effect to an entity with the given parameters
func (s *EquipmentSystem) AddItemEffect(entityID ecs.EntityID, effect components.GameEffect) error {
	// Get or create the equipment component
	equipComp, exists := s.world.GetComponent(entityID, components.Equipment)
	if !exists {
		equipComp = &components.EquipmentComponent{
			EquippedItems: make(map[components.EquipmentSlot]ecs.EntityID),
			ActiveEffects: make(map[ecs.EntityID][]components.GameEffect),
		}
		s.world.AddComponent(entityID, components.Equipment, equipComp)
	}

	equipment, ok := equipComp.(*components.EquipmentComponent)
	if !ok {
		return fmt.Errorf("invalid Equipment component type")
	}

	// Add the effect
	equipment.AddEffect(effect.Source, effect)
	return nil
}

// RemoveItemEffect removes an effect from an entity
func (s *EquipmentSystem) RemoveItemEffect(entityID ecs.EntityID, effect components.GameEffect) error {
	if comp, exists := s.world.GetComponent(entityID, components.Equipment); exists {
		if equipComp, ok := comp.(components.EquipmentComponent); ok {
			equipComp.RemoveEffects(effect.Source)
		}
	}
	return nil
}

// GetInverseEffect creates an inverse effect for removal
func (s *EquipmentSystem) GetInverseEffect(effect components.GameEffect) (components.GameEffect, error) {
	// For now, we'll just return a copy with the same parameters
	// In the future, we might want to handle different operation types differently
	return effect, nil
}

// Helper function to get the name of an item
func (s *EquipmentSystem) getItemName(world *ecs.World, itemID ecs.EntityID) string {
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		return nameComp.(*components.NameComponent).Name
	}
	return "unknown item"
}
