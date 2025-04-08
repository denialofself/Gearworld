package systems

import (
	"fmt"
	"math"
	"math/rand"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// Define a constant for our AIPathEvent type
const EventAIPath ecs.EventType = "ai_path_event"

// AITurnProcessorSystem handles AI movement based on calculated paths
type AITurnProcessorSystem struct{}

// Define action costs
const (
	MoveCost   = 2
	WaitCost   = 1
	AttackCost = 3
)

// NewAITurnProcessorSystem creates a new AI turn processor system
func NewAITurnProcessorSystem() *AITurnProcessorSystem {
	return &AITurnProcessorSystem{}
}

// Initialize sets up event listeners for the AI turn processor system
func (s *AITurnProcessorSystem) Initialize(world *ecs.World) {
	// Subscribe to AI path events
	world.GetEventManager().Subscribe(EventAIPath, func(event ecs.Event) {
		s.HandlePathEvent(world, event)
	})
}

// HandlePathEvent processes AI path events
func (s *AITurnProcessorSystem) HandlePathEvent(world *ecs.World, event ecs.Event) {
	if pathEvent, ok := event.(AIPathEvent); ok {
		entityID := pathEvent.EntityID
		path := pathEvent.Path

		// Get AI component
		aiComp, hasAI := world.GetComponent(entityID, components.AI)
		if !hasAI {
			return
		}
		ai := aiComp.(*components.AIComponent)

		// Get position component
		posComp, hasPos := world.GetComponent(entityID, components.Position)
		if !hasPos {
			return
		}
		pos := posComp.(*components.PositionComponent)

		// Get stats component for recovery value
		statsComp, hasStats := world.GetComponent(entityID, components.Stats)
		var recoveryPoints int
		if hasStats {
			stats := statsComp.(*components.StatsComponent)
			recoveryPoints = stats.Recovery
		} else {
			recoveryPoints = 1 // Default recovery
		}

		// Process movement based on path
		s.processTurn(world, uint64(entityID), ai, pos, path, recoveryPoints)
	}
}

// Update doesn't need to do anything since we work through event handling
func (s *AITurnProcessorSystem) Update(world *ecs.World, dt float64) {
	// The system is event-driven, no need for regular updates
}

// isAdjacentToPlayer checks if the given position is adjacent to the player
func (s *AITurnProcessorSystem) isAdjacentToPlayer(world *ecs.World, x, y int) (bool, ecs.EntityID) {
	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return false, 0
	}

	playerID := playerEntities[0].ID
	playerPos, hasPos := world.GetComponent(playerID, components.Position)
	if !hasPos {
		return false, 0
	}

	pos := playerPos.(*components.PositionComponent)

	// Check if adjacent (including diagonals)
	dx := int(math.Abs(float64(pos.X - x)))
	dy := int(math.Abs(float64(pos.Y - y)))

	// Check if player is adjacent (distance of 1 in either or both directions)
	if dx <= 1 && dy <= 1 && !(dx == 0 && dy == 0) {
		return true, playerID
	}

	return false, 0
}

// processTurn handles AI turn processing
func (s *AITurnProcessorSystem) processTurn(world *ecs.World, entityID uint64, ai *components.AIComponent, pos *components.PositionComponent, path []components.PathNode, recoveryPoints int) {
	// Get stats component for action points
	statsComp, hasStats := world.GetComponent(ecs.EntityID(entityID), components.Stats)
	if !hasStats {
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Entity %d has no stats component, cannot process turn", entityID))
		return
	}
	stats := statsComp.(*components.StatsComponent)
	// Check if we're adjacent to the player and can attack
	if adjacent, playerID := s.isAdjacentToPlayer(world, pos.X, pos.Y); adjacent && stats.ActionPoints >= AttackCost {
		// Process attack based on AI type
		switch ai.Type {
		case "slow_chase":
			// Slow chase type always attacks when adjacent
			world.GetEventManager().Emit(EnemyAttackEvent{
				AttackerID: ecs.EntityID(entityID),
				TargetID:   playerID,
				X:          pos.X,
				Y:          pos.Y,
			})
			stats.ActionPoints -= AttackCost
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AI attacked player (AP: %d)", stats.ActionPoints))
			return
		}
	}

	// Process movement or waiting based on action points and path
	if len(path) > 0 {
		// Get the next step in the path
		nextStep := path[0]
		GetMessageLog().Add(fmt.Sprintf("DEBUG: AI turn processor - Next step: %d,%d, AP: %d", nextStep.X, nextStep.Y, stats.ActionPoints))

		// Check if we can move there
		canMove := s.isValidMove(world, nextStep.X, nextStep.Y)

		if canMove && stats.ActionPoints >= MoveCost {
			// Handle AI type specific movement
			switch ai.Type {
			case "slow_chase":
				// 1 in 6 chance to skip movement
				if rand.Intn(6) == 0 {
					GetMessageLog().Add(fmt.Sprintf("DEBUG: Slow chase AI skipped movement"))
					stats.ActionPoints -= WaitCost
					return
				}
			}

			// Move to the next step
			oldX, oldY := pos.X, pos.Y
			pos.X = nextStep.X
			pos.Y = nextStep.Y

			// Consume action points
			stats.ActionPoints -= MoveCost

			// Emit movement event
			world.EmitEvent(EntityMoveEvent{
				EntityID: ecs.EntityID(entityID),
				FromX:    oldX,
				FromY:    oldY,
				ToX:      pos.X,
				ToY:      pos.Y,
			})
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AI moved from %d,%d to %d,%d (AP: %d)", oldX, oldY, pos.X, pos.Y, stats.ActionPoints))
		} else if stats.ActionPoints >= WaitCost {
			// Can't move but can wait (might be blocked by another entity)
			stats.ActionPoints -= WaitCost
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AI waiting (AP: %d)", stats.ActionPoints))
		} else {
			// Not enough action points, recover
			stats.ActionPoints += recoveryPoints
			if stats.ActionPoints > stats.MaxActionPoints {
				stats.ActionPoints = stats.MaxActionPoints
			}
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AI recovering %d points (AP: %d)", recoveryPoints, stats.ActionPoints))
		}
	} else {
		// No path, just recover action points
		stats.ActionPoints += recoveryPoints
		if stats.ActionPoints > stats.MaxActionPoints {
			stats.ActionPoints = stats.MaxActionPoints
		}
		GetMessageLog().Add(fmt.Sprintf("DEBUG: AI has no path, recovering %d points (AP: %d)", recoveryPoints, stats.ActionPoints))
	}
}

// isValidMove checks if a position is a valid movement destination
func (s *AITurnProcessorSystem) isValidMove(world *ecs.World, x, y int) bool {
	// Get map for checking walls
	mapEntities := world.GetEntitiesWithTag("map")
	if len(mapEntities) == 0 {
		return false
	}

	mapComp, exists := world.GetComponent(mapEntities[0].ID, components.MapComponentID)
	if !exists {
		return false
	}
	gameMap := mapComp.(*components.MapComponent)

	// Check for walls
	if gameMap.IsWall(x, y) {
		return false
	}

	// Check for entity collision
	for _, entity := range world.GetAllEntities() {
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		if !hasPos {
			continue
		}

		pos := posComp.(*components.PositionComponent)
		if pos.X == x && pos.Y == y {
			// Position is occupied by an entity, check if it blocks
			if collComp, hasCol := world.GetComponent(entity.ID, components.Collision); hasCol {
				collision := collComp.(*components.CollisionComponent)
				if collision.Blocks {
					return false
				}
			}
		}
	}

	return true
}
