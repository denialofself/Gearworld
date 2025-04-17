package systems

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// CombatSystem handles combat interactions between entities
type CombatSystem struct {
	initialized bool
}

// NewCombatSystem creates a new combat system
func NewCombatSystem() *CombatSystem {
	return &CombatSystem{}
}

// Initialize sets up event listeners
func (s *CombatSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	// Subscribe to collision events
	world.GetEventManager().Subscribe(EventCollision, func(event ecs.Event) {
		collisionEvent := event.(CollisionEvent)
		s.handleCollision(world, collisionEvent)
	})

	// Subscribe to enemy attack events
	world.GetEventManager().Subscribe(EventEnemyAttack, func(event ecs.Event) {
		attackEvent := event.(EnemyAttackEvent)
		s.handleEnemyAttack(world, attackEvent)
	})

	s.initialized = true
}

// handleCollision processes a collision event
func (s *CombatSystem) handleCollision(world *ecs.World, event CollisionEvent) {
	// Check if this collision should trigger combat
	entityID1 := event.EntityID1
	entityID2 := event.EntityID2

	// Check if either entity is the player and the other is an enemy
	isPlayerInvolved := isPlayer(world, entityID1) || isPlayer(world, entityID2)

	if isPlayerInvolved {
		// Check if both entities are on the same map
		mapID1 := getEntityMapID(world, entityID1)
		mapID2 := getEntityMapID(world, entityID2)

		if mapID1 != mapID2 {
			GetDebugLog().Add(fmt.Sprintf("DEBUG: Collision combat prevented - entities on different maps (%d vs %d)", mapID1, mapID2))
			return
		}

		// Determine attacker and defender
		var attackerID, defenderID ecs.EntityID
		if isPlayer(world, entityID1) {
			attackerID = entityID1
			defenderID = entityID2
		} else {
			attackerID = entityID2
			defenderID = entityID1
		}

		// Process combat
		s.ProcessCombat(world, attackerID, defenderID)
	}
}

// handleEnemyAttack processes an enemy attack event
func (s *CombatSystem) handleEnemyAttack(world *ecs.World, event EnemyAttackEvent) {
	// We already know the attacker and defender from the event
	attackerID := event.AttackerID
	defenderID := event.TargetID

	// Check if both entities are on the same map
	attackerMapID := getEntityMapID(world, attackerID)
	defenderMapID := getEntityMapID(world, defenderID)

	if attackerMapID != defenderMapID {
		GetDebugLog().Add(fmt.Sprintf("DEBUG: Combat prevented - entities on different maps (attacker: %d, defender: %d)", attackerMapID, defenderMapID))
		return
	}

	// Process the combat with the enemy as the attacker
	s.ProcessCombat(world, attackerID, defenderID)
}

// getEntityMapID returns the map ID an entity is on, or 0 if not on a map
func getEntityMapID(world *ecs.World, entityID ecs.EntityID) ecs.EntityID {
	if world.HasComponent(entityID, components.MapContextID) {
		mapContextComp, _ := world.GetComponent(entityID, components.MapContextID)
		mapContext := mapContextComp.(*components.MapContextComponent)
		return mapContext.MapID
	}
	return 0
}

// Update registers with event system if not already initialized
func (s *CombatSystem) Update(world *ecs.World, dt float64) {
	// Ensure system is initialized with event handlers
	if !s.initialized {
		s.Initialize(world)
	}
}

// ProcessCombat handles combat between an attacker and defender
func (s *CombatSystem) ProcessCombat(world *ecs.World, attackerID, defenderID ecs.EntityID) bool {
	// Get attacker stats
	attackerStatsComp, hasAttackerStats := world.GetComponent(attackerID, components.Stats)
	if !hasAttackerStats {
		return false
	}
	attackerStats := attackerStatsComp.(*components.StatsComponent)

	// Get defender stats
	defenderStatsComp, hasDefenderStats := world.GetComponent(defenderID, components.Stats)
	if !hasDefenderStats {
		return false
	}
	defenderStats := defenderStatsComp.(*components.StatsComponent)

	// Get entity names or descriptions for the message log
	attackerName := getEntityName(world, attackerID)
	defenderName := getEntityName(world, defenderID)

	// Roll d20 and add attacker's attack bonus
	d20Roll := rand.Intn(20) + 1 // 1-20
	attackRoll := d20Roll + attackerStats.Attack

	// Calculate damage (attack roll minus defender's defense)
	damage := attackRoll - defenderStats.Defense
	// Log the attack roll
	rollMsg := fmt.Sprintf("%s attacks %s! (Roll: %d + %d = %d)",
		attackerName, defenderName, d20Roll, attackerStats.Attack, attackRoll)
	GetMessageLog().AddCombat(rollMsg)

	// Handle the outcome
	if damage <= 0 {
		// Attack missed or was blocked
		GetMessageLog().AddCombat(fmt.Sprintf("%s's attack was ineffective!", attackerName))
		return false
	} else {
		// Apply damage
		defenderStats.Health -= damage
		damageMsg := fmt.Sprintf("%s hit %s for %d damage! %s has %d/%d HP remaining.",
			attackerName, defenderName, damage, defenderName, defenderStats.Health, defenderStats.MaxHealth)
		GetMessageLog().AddCombat(damageMsg)

		// Check if defender is defeated
		if defenderStats.Health <= 0 {
			GetMessageLog().AddAlert(fmt.Sprintf("%s was defeated!", defenderName))

			// Emit death event before handling the entity
			world.GetEventManager().Emit(DeathEvent{
				EntityID: defenderID,
				KillerID: attackerID,
			})

			// Handle player death
			if isPlayer(world, defenderID) {
				GetMessageLog().AddAlert("Game Over! You died.")
				// Could trigger game over state here
			} else {
				// Remove the defeated entity
				world.RemoveEntity(defenderID)
			}
		}

		return true
	}
}

// Helper function to get an entity's name or description
func getEntityName(world *ecs.World, entityID ecs.EntityID) string {
	if isPlayer(world, entityID) {
		return "Player"
	}

	// Try to get name component first
	if nameComp, hasName := world.GetComponent(entityID, components.Name); hasName {
		name := nameComp.(*components.NameComponent)
		return name.Name
	}

	// Fallback to AI type if no name component
	if aiComp, hasAI := world.GetComponent(entityID, components.AI); hasAI {
		ai := aiComp.(*components.AIComponent)
		return capitalizeFirstLetter(ai.Type)
	}

	// Last fallback
	return "Entity #" + strconv.FormatUint(uint64(entityID), 10)
}

// Helper function to check if an entity is the player
func isPlayer(world *ecs.World, entityID ecs.EntityID) bool {
	// Get all entities with the "player" tag
	playerEntities := world.GetEntitiesWithTag("player")

	// Check if any of them match our entity ID
	for _, entity := range playerEntities {
		if entity.ID == entityID {
			return true
		}
	}

	return false
}

// Helper function to capitalize the first letter of a string
func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
