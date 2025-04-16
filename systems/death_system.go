package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
)

// DeathSystem handles death events and their consequences
type DeathSystem struct {
	initialized bool
}

// NewDeathSystem creates a new death system
func NewDeathSystem() *DeathSystem {
	return &DeathSystem{}
}

// Initialize sets up event listeners
func (s *DeathSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	// Subscribe to death events
	world.GetEventManager().Subscribe(EventDeath, func(event ecs.Event) {
		deathEvent := event.(DeathEvent)
		s.handleDeath(world, deathEvent)
	})

	s.initialized = true
}

// handleDeath processes a death event
func (s *DeathSystem) handleDeath(world *ecs.World, event DeathEvent) {
	// Get entity names for logging
	entityName := getEntityName(world, event.EntityID)
	killerName := getEntityName(world, event.KillerID)

	// Log the death
	GetMessageLog().AddAlert(fmt.Sprintf("%s was killed by %s!", entityName, killerName))

	// If the player died, emit game over event
	if isPlayer(world, event.EntityID) {
		GetMessageLog().AddAlert("Game Over! You were defeated.")
		world.GetEventManager().Emit(GameOverEvent{PlayerID: event.EntityID})
	} else if isPlayer(world, event.KillerID) {
		// Player killed something - check for XP gain
		if monsterStatsComp, hasMonsterStats := world.GetComponent(event.EntityID, components.Stats); hasMonsterStats {
			monsterStats := monsterStatsComp.(*components.StatsComponent)
			// Get the player's stats component
			if playerStatsComp, hasPlayerStats := world.GetComponent(event.KillerID, components.Stats); hasPlayerStats {
				playerStats := playerStatsComp.(*components.StatsComponent)
				// Add XP to player
				playerStats.Exp += monsterStats.Exp
				GetMessageLog().AddAlert(fmt.Sprintf("You gained %d XP!", monsterStats.Exp))
			}
		}
	}
}

// Update registers with event system if not already initialized
func (s *DeathSystem) Update(world *ecs.World, dt float64) {
	// Ensure system is initialized with event handlers
	if !s.initialized {
		s.Initialize(world)
	}
}
