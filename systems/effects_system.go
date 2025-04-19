package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// EffectsSystem handles all types of effects in a unified way
type EffectsSystem struct {
	initialized bool
	world       *ecs.World
}

// NewEffectsSystem creates a new effects system
func NewEffectsSystem() *EffectsSystem {
	return &EffectsSystem{
		initialized: false,
	}
}

// Initialize sets up the effects system
func (s *EffectsSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	s.world = world

	// Subscribe to equipment events
	world.GetEventManager().Subscribe("item_equipped", func(event ecs.Event) {
		if equipEvent, ok := event.(ItemEquippedEvent); ok {
			s.HandleItemEquipped(world, equipEvent.EntityID, equipEvent.ItemID)
		}
	})

	world.GetEventManager().Subscribe("item_unequipped", func(event ecs.Event) {
		if unequipEvent, ok := event.(ItemUnequippedEvent); ok {
			s.HandleItemUnequipped(world, unequipEvent.EntityID, unequipEvent.ItemID)
		}
	})

	// Subscribe to effects events
	world.GetEventManager().Subscribe(EventEffects, func(event ecs.Event) {
		if effectEvent, ok := event.(EffectsEvent); ok {
			// Get the item component to access its effects
			if itemComp, exists := world.GetComponent(effectEvent.Source, components.Item); exists {
				if item, ok := itemComp.(*components.ItemComponent); ok {
					if effects, ok := item.Data.([]components.GameEffect); ok {
						// Get or create the effect component to track applied effects
						effectComp, exists := world.GetComponent(effectEvent.EntityID, components.Effect)
						if !exists {
							effectComp = &components.EffectComponent{
								Effects: make([]components.GameEffect, 0),
							}
							world.AddComponent(effectEvent.EntityID, components.Effect, effectComp)
						}
						effectComponent := effectComp.(*components.EffectComponent)

						// Check if these effects have already been applied
						alreadyApplied := false
						for _, existing := range effectComponent.Effects {
							if existing.Source == effectEvent.Source {
								alreadyApplied = true
								break
							}
						}

						if !alreadyApplied {
							// Apply the item's effects
							s.ApplyEntityEffects(world, effectEvent.EntityID, effects)
						}
					}
				}
			}
		}
	})

	// Subscribe to turn completed events
	world.GetEventManager().Subscribe("turn_completed", func(event ecs.Event) {
		if _, ok := event.(TurnCompletedEvent); ok {
			// Process effects for all entities with the Effect component
			for _, entity := range world.GetEntitiesWithComponent(components.Effect) {
				s.ProcessEffects(world, entity.ID)
			}
		}
	})

	s.initialized = true
}

// Update ensures the system is initialized but doesn't process effects every frame
func (s *EffectsSystem) Update(world *ecs.World, dt float64) {
	if !s.initialized {
		s.Initialize(world)
	}
}

// ApplyEntityEffects applies a list of effects to an entity
func (s *EffectsSystem) ApplyEntityEffects(world *ecs.World, entityID ecs.EntityID, effects []components.GameEffect) error {
	entity := world.GetEntity(entityID)
	if entity == nil {
		return nil
	}

	// Get or create the effect component
	effectComp, exists := world.GetComponent(entityID, components.Effect)
	if !exists {
		effectComp = &components.EffectComponent{
			Effects: make([]components.GameEffect, 0),
		}
		world.AddComponent(entityID, components.Effect, effectComp)
	}
	effectComponent := effectComp.(*components.EffectComponent)

	// Get the stats component for applying effects
	statsComp, exists := world.GetComponent(entityID, components.Stats)
	if !exists {
		return nil
	}
	stats := statsComp.(*components.StatsComponent)

	// Log the effects being applied
	GetDebugLog().Add(fmt.Sprintf("Applying %d effects to entity %d:", len(effects), entityID))
	for _, effect := range effects {
		GetDebugLog().Add(fmt.Sprintf("  - Effect: %s %s %v on %s.%s",
			effect.Type, effect.Operation, effect.Value,
			effect.Target.Component, effect.Target.Property))
	}

	// Log current stats before effects
	GetDebugLog().Add(fmt.Sprintf("Current stats before effects:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Apply each effect
	for _, effect := range effects {
		// Check for duplicate effects
		isDuplicate := false
		for i, existing := range effectComponent.Effects {
			if existing.Type == effect.Type &&
				existing.Operation == effect.Operation &&
				existing.Target.Component == effect.Target.Component &&
				existing.Target.Property == effect.Target.Property &&
				existing.Source == effect.Source {
				// Update existing effect
				effectComponent.Effects[i] = effect
				isDuplicate = true
				GetDebugLog().Add(fmt.Sprintf("  - Updated existing effect"))
				break
			}
		}

		if !isDuplicate {
			// Add new effect
			effectComponent.Effects = append(effectComponent.Effects, effect)
			GetDebugLog().Add(fmt.Sprintf("  - Added new effect"))
		}

		// Don't apply instant effects here since they are handled by the event system
		// if effect.Type == components.EffectTypeInstant && !isDuplicate {
		// 	s.applyEffect(world, entityID, effect)
		// }
	}

	// Log stats after effects
	GetDebugLog().Add(fmt.Sprintf("Stats after effects:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	return nil
}

// RemoveEntityEffects removes a list of effects from an entity
func (s *EffectsSystem) RemoveEntityEffects(world *ecs.World, entityID ecs.EntityID, effects []components.GameEffect) error {
	entity := world.GetEntity(entityID)
	if entity == nil {
		return nil
	}

	// Get the effect component
	effectComp, exists := world.GetComponent(entityID, components.Effect)
	if !exists {
		return nil
	}
	effectComponent := effectComp.(*components.EffectComponent)

	// Get the stats component for removing effects
	statsComp, exists := world.GetComponent(entityID, components.Stats)
	if !exists {
		return nil
	}
	stats := statsComp.(*components.StatsComponent)

	// Log the effects being removed
	GetDebugLog().Add(fmt.Sprintf("Removing %d effects from entity %d:", len(effects), entityID))
	for _, effect := range effects {
		GetDebugLog().Add(fmt.Sprintf("  - Effect: %s %s %v on %s.%s",
			effect.Type, effect.Operation, effect.Value,
			effect.Target.Component, effect.Target.Property))
	}

	// Log current stats before removal
	GetDebugLog().Add(fmt.Sprintf("Current stats before removal:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Remove each effect
	for _, effect := range effects {
		for i := len(effectComponent.Effects) - 1; i >= 0; i-- {
			existing := effectComponent.Effects[i]
			if existing.Type == effect.Type &&
				existing.Operation == effect.Operation &&
				existing.Target.Component == effect.Target.Component &&
				existing.Target.Property == effect.Target.Property &&
				existing.Source == effect.Source {
				// Remove the effect
				effectComponent.Effects = append(effectComponent.Effects[:i], effectComponent.Effects[i+1:]...)
				GetDebugLog().Add(fmt.Sprintf("  - Removed effect"))
				break
			}
		}
	}

	// Log stats after removal
	GetDebugLog().Add(fmt.Sprintf("Stats after removal:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	return nil
}

// CreateGameEffect creates a new effect with the given parameters
func (s *EffectsSystem) CreateGameEffect(effectType components.EffectType, operation components.EffectOperation, value interface{}, duration int, source ecs.EntityID, targetComponent string, targetProperty string) components.GameEffect {
	return components.GameEffect{
		Type:      effectType,
		Operation: operation,
		Value:     value,
		Duration:  duration,
		Source:    source,
		Target: struct {
			Component string
			Property  string
		}{
			Component: targetComponent,
			Property:  targetProperty,
		},
	}
}

// ProcessEffects processes all active effects on an entity
func (s *EffectsSystem) ProcessEffects(world *ecs.World, entityID ecs.EntityID) {
	if comp, exists := world.GetComponent(entityID, components.Effect); exists {
		if effectComp, ok := comp.(*components.EffectComponent); ok {
			// Create a new slice to store effects that should remain
			remainingEffects := make([]components.GameEffect, 0)

			for _, effect := range effectComp.Effects {
				switch effect.Type {
				case components.EffectTypeInstant:
					// Apply instant effect and don't keep it
					s.applyEffect(world, entityID, effect)

				case components.EffectTypeDuration:
					// Apply effect and keep it if duration remains
					s.applyEffect(world, entityID, effect)
					if effect.Duration > 0 {
						effect.Duration--
						remainingEffects = append(remainingEffects, effect)
					}

				case components.EffectTypePeriodic:
					// Apply periodic effect and keep it if duration remains
					GetDebugLog().Add(fmt.Sprintf("Processing periodic effect on entity %d - Duration: %d", entityID, effect.Duration))
					s.applyEffect(world, entityID, effect)
					if effect.Duration > 0 {
						effect.Duration--
						remainingEffects = append(remainingEffects, effect)
						GetDebugLog().Add(fmt.Sprintf("Keeping periodic effect, new duration: %d", effect.Duration))
					} else {
						GetDebugLog().Add("Removing periodic effect - duration expired")
					}

				case components.EffectTypeConditional:
					// Keep conditional effects
					remainingEffects = append(remainingEffects, effect)
				}
			}

			// Update the effects list
			effectComp.Effects = remainingEffects
		}
	}
}

// applyEffect applies a single effect to an entity
func (s *EffectsSystem) applyEffect(world *ecs.World, entityID ecs.EntityID, effect components.GameEffect) {
	// Get the target component based on the effect's target info
	var componentID ecs.ComponentID
	switch effect.Target.Component {
	case "Stats":
		componentID = components.Stats
	case "FOV":
		componentID = components.FOV
	default:
		GetMessageLog().Add(fmt.Sprintf("Unknown component type: %s", effect.Target.Component))
		return
	}

	if comp, exists := world.GetComponent(entityID, componentID); exists {
		switch effect.Target.Component {
		case "Stats":
			if stats, ok := comp.(*components.StatsComponent); ok {
				// Calculate the effect value, handling dice roll notation
				value := s.calculateEffectValue(effect.Value)

				// Apply effect based on the target property
				switch effect.Target.Property {
				case "Health":
					switch effect.Operation {
					case components.EffectOpAdd:
						stats.Health += int(value)
						// Cap health at max health
						if stats.Health > stats.MaxHealth {
							stats.Health = stats.MaxHealth
						}
					case components.EffectOpSubtract:
						stats.Health -= int(value)
						if stats.Health < 0 {
							stats.Health = 0
						}
					case components.EffectOpMultiply:
						stats.Health = int(float64(stats.Health) * value)
					case components.EffectOpSet:
						stats.Health = int(value)
					}
				case "Attack":
					switch effect.Operation {
					case components.EffectOpAdd:
						stats.Attack += int(value)
					case components.EffectOpSubtract:
						stats.Attack -= int(value)
					case components.EffectOpMultiply:
						stats.Attack = int(float64(stats.Attack) * value)
					case components.EffectOpSet:
						stats.Attack = int(value)
					}
				case "Defense":
					switch effect.Operation {
					case components.EffectOpAdd:
						stats.Defense += int(value)
					case components.EffectOpSubtract:
						stats.Defense -= int(value)
					case components.EffectOpMultiply:
						stats.Defense = int(float64(stats.Defense) * value)
					case components.EffectOpSet:
						stats.Defense = int(value)
					}
				case "MaxHealth":
					switch effect.Operation {
					case components.EffectOpAdd:
						stats.MaxHealth += int(value)
						// Also increase current health if max health increases
						stats.Health += int(value)
					case components.EffectOpSubtract:
						stats.MaxHealth -= int(value)
						// Cap current health at new max health
						if stats.Health > stats.MaxHealth {
							stats.Health = stats.MaxHealth
						}
					case components.EffectOpMultiply:
						stats.MaxHealth = int(float64(stats.MaxHealth) * value)
						// Adjust current health proportionally
						stats.Health = int(float64(stats.Health) * value)
					case components.EffectOpSet:
						stats.MaxHealth = int(value)
						// Cap current health at new max health
						if stats.Health > stats.MaxHealth {
							stats.Health = stats.MaxHealth
						}
					}
				}
			}
		case "FOV":
			if fov, ok := comp.(*components.FOVComponent); ok {
				// Calculate the effect value, handling dice roll notation
				value := s.calculateEffectValue(effect.Value)

				switch effect.Target.Property {
				case "Range":
					switch effect.Operation {
					case components.EffectOpAdd:
						fov.Range += int(value)
					case components.EffectOpSubtract:
						fov.Range -= int(value)
					case components.EffectOpMultiply:
						fov.Range = int(float64(fov.Range) * value)
					case components.EffectOpSet:
						fov.Range = int(value)
					}
				case "LightRange":
					switch effect.Operation {
					case components.EffectOpAdd:
						fov.LightRange += int(value)
						fov.LightSource = fov.LightRange > 0
					case components.EffectOpSubtract:
						fov.LightRange -= int(value)
						fov.LightSource = fov.LightRange > 0
					case components.EffectOpMultiply:
						fov.LightRange = int(float64(fov.LightRange) * value)
						fov.LightSource = fov.LightRange > 0
					case components.EffectOpSet:
						fov.LightRange = int(value)
						fov.LightSource = fov.LightRange > 0
					}
				}
			}
		}
	}
}

// calculateEffectValue calculates the effect value, handling dice roll notation
func (s *EffectsSystem) calculateEffectValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case string:
		// Handle dice roll notation (e.g., "1d4", "2d6", etc.)
		if strings.Contains(v, "d") {
			parts := strings.Split(v, "d")
			if len(parts) != 2 {
				return 0
			}
			numDice, err1 := strconv.Atoi(parts[0])
			diceSize, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				return 0
			}
			var total int
			for i := 0; i < numDice; i++ {
				total += rand.Intn(diceSize) + 1
			}
			return float64(total)
		}
		// Try to parse as a regular number
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
	}
	return 0
}

// HandleItemEquipped processes equipment effects when an item is equipped
func (s *EffectsSystem) HandleItemEquipped(world *ecs.World, entityID ecs.EntityID, itemID ecs.EntityID) error {
	// Get the item's effects
	itemComp, exists := world.GetComponent(itemID, components.Item)
	if !exists {
		return fmt.Errorf("item %d does not have an Item component", itemID)
	}

	item, ok := itemComp.(*components.ItemComponent)
	if !ok {
		return fmt.Errorf("item %d has invalid Item component type", itemID)
	}

	effects, ok := item.Data.([]components.GameEffect)
	if !ok {
		return fmt.Errorf("item %d has invalid effects data", itemID)
	}

	// Get the stats component for applying effects
	statsComp, exists := world.GetComponent(entityID, components.Stats)
	if !exists {
		return fmt.Errorf("entity %d does not have a Stats component", entityID)
	}
	stats := statsComp.(*components.StatsComponent)

	// Get the effect component to track applied effects
	effectComp, exists := world.GetComponent(entityID, components.Effect)
	if !exists {
		effectComp = &components.EffectComponent{
			Effects: make([]components.GameEffect, 0),
		}
		world.AddComponent(entityID, components.Effect, effectComp)
	}
	effectComponent := effectComp.(*components.EffectComponent)

	// Log current stats before effects
	GetDebugLog().Add(fmt.Sprintf("Current stats before equip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Apply each effect only if it hasn't been applied yet
	for _, effect := range effects {
		if effect.Type == components.EffectTypeEquipment {
			// Check if this effect is already applied
			isDuplicate := false
			for _, existing := range effectComponent.Effects {
				if existing.Type == effect.Type &&
					existing.Operation == effect.Operation &&
					existing.Target.Component == effect.Target.Component &&
					existing.Target.Property == effect.Target.Property &&
					existing.Source == effect.Source {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				s.applyEffect(world, entityID, effect)
				// Add the effect to the component to track it
				effectComponent.Effects = append(effectComponent.Effects, effect)
			}
		}
	}

	// Log stats after effects
	GetDebugLog().Add(fmt.Sprintf("Stats after equip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	return nil
}

// HandleItemUnequipped processes equipment effects when an item is unequipped
func (s *EffectsSystem) HandleItemUnequipped(world *ecs.World, entityID ecs.EntityID, itemID ecs.EntityID) error {
	// Get the item's effects
	itemComp, exists := world.GetComponent(itemID, components.Item)
	if !exists {
		return fmt.Errorf("item %d does not have an Item component", itemID)
	}

	item, ok := itemComp.(*components.ItemComponent)
	if !ok {
		return fmt.Errorf("item %d has invalid Item component type", itemID)
	}

	effects, ok := item.Data.([]components.GameEffect)
	if !ok {
		return fmt.Errorf("item %d has invalid effects data", itemID)
	}

	// Get the stats component for removing effects
	statsComp, exists := world.GetComponent(entityID, components.Stats)
	if !exists {
		return fmt.Errorf("entity %d does not have a Stats component", entityID)
	}
	stats := statsComp.(*components.StatsComponent)

	// Log current stats before removal
	GetDebugLog().Add(fmt.Sprintf("Current stats before unequip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	// Remove each effect
	for _, effect := range effects {
		if effect.Type == components.EffectTypeEquipment {
			// Invert the operation to remove the effect
			invertedEffect := effect
			switch effect.Operation {
			case components.EffectOpAdd:
				invertedEffect.Operation = components.EffectOpSubtract
			case components.EffectOpSubtract:
				invertedEffect.Operation = components.EffectOpAdd
			case components.EffectOpMultiply:
				// For multiply operations, we'll use subtract since we don't have divide
				invertedEffect.Operation = components.EffectOpSubtract
				// Adjust the value to approximate the inverse of multiplication
				if val, ok := invertedEffect.Value.(float64); ok {
					invertedEffect.Value = val - 1
				}
			case components.EffectOpSet:
				// For set operations, we need to restore the original value
				// This would require storing the original value somewhere
				// For now, we'll just skip set operations
				continue
			}
			s.applyEffect(world, entityID, invertedEffect)
		}
	}

	// Log stats after removal
	GetDebugLog().Add(fmt.Sprintf("Stats after unequip:"))
	GetDebugLog().Add(fmt.Sprintf("  - Health: %d/%d", stats.Health, stats.MaxHealth))
	GetDebugLog().Add(fmt.Sprintf("  - Attack: %d", stats.Attack))
	GetDebugLog().Add(fmt.Sprintf("  - Defense: %d", stats.Defense))

	return nil
}
