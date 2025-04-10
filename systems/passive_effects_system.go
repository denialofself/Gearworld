package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"strconv"
)

// PassiveEffectsSystem handles passive effects like healing when resting
type PassiveEffectsSystem struct {
	initialized bool
}

// NewPassiveEffectsSystem creates a new passive effects system
func NewPassiveEffectsSystem() *PassiveEffectsSystem {
	return &PassiveEffectsSystem{}
}

// Initialize sets up event listeners for the passive effects system
func (s *PassiveEffectsSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	// Subscribe to rest events
	world.GetEventManager().Subscribe(EventRest, func(event ecs.Event) {
		restEvent := event.(RestEvent)
		s.handleRest(world, restEvent)
	})

	s.initialized = true
}

// Update ensures the system is initialized
func (s *PassiveEffectsSystem) Update(world *ecs.World, dt float64) {
	// Ensure system is initialized with event handlers
	if !s.initialized {
		s.Initialize(world)
	}
}

// handleRest processes rest events and applies healing
func (s *PassiveEffectsSystem) handleRest(world *ecs.World, event RestEvent) {
	entityID := event.EntityID

	// Get stats component
	statsComp, hasStats := world.GetComponent(entityID, components.Stats)
	if !hasStats {
		return
	}

	stats := statsComp.(*components.StatsComponent)
	
	// Calculate healing amount based on healing factor
	healAmount := stats.HealingFactor
	
	// Apply healing (don't exceed max health)
	oldHealth := stats.Health
	stats.Health += healAmount
	if stats.Health > stats.MaxHealth {
		stats.Health = stats.MaxHealth
	}

	// Only log if healing actually occurred
	if stats.Health > oldHealth {
		actualHealing := stats.Health - oldHealth
		// Log the healing
		GetMessageLog().Add("You rest for a moment and recover " + strconv.Itoa(actualHealing) + " health.")
	} else if stats.Health == stats.MaxHealth {
		GetMessageLog().Add("You rest for a moment, but you're already at full health.")
	}
}
