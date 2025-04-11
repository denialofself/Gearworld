package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// MovementSystem handles entity movement
type MovementSystem struct {
	// Flags to track internal states
	moveAttempted bool // Tracks if a move attempt has been processed this frame
}

// NewMovementSystem creates a new movement system
func NewMovementSystem() *MovementSystem {
	return &MovementSystem{
		moveAttempted: false,
	}
}

// Initialize sets up the event listeners for the movement system
func (s *MovementSystem) Initialize(world *ecs.World) {
	// Register to listen for movement attempt events
	world.RegisterEventListener(s.handleMoveAttempt)
}

// handleMoveAttempt processes movement attempt events
func (s *MovementSystem) handleMoveAttempt(world *ecs.World, event interface{}) {
	// Only process PlayerMoveAttemptEvent
	moveAttempt, ok := event.(PlayerMoveAttemptEvent)
	if !ok {
		return
	}

	// Set flag that a movement attempt was processed
	s.moveAttempted = true

	// Get the active map from the map registry system
	var activeMapID ecs.EntityID
	for _, system := range world.GetSystems() {
		if mapReg, ok := system.(interface{ GetActiveMap() *ecs.Entity }); ok {
			if activeMap := mapReg.GetActiveMap(); activeMap != nil {
				activeMapID = activeMap.ID
				break
			}
		}
	}

	if activeMapID == 0 {
		// No active map found, fail silently
		return
	}

	// Check if the move is valid
	canMove := s.isValidMoveStandard(world, activeMapID, moveAttempt.ToX, moveAttempt.ToY, moveAttempt.EntityID)

	// If move is valid, update position
	if canMove {
		// Get the entity's position component
		posComp, exists := world.GetComponent(moveAttempt.EntityID, components.Position)
		if !exists {
			return
		}
		position := posComp.(*components.PositionComponent)

		// Store the old position
		oldX, oldY := position.X, position.Y

		// Update position
		position.X = moveAttempt.ToX
		position.Y = moveAttempt.ToY

		// Emit movement event
		world.EmitEvent(PlayerMoveEvent{
			EntityID: moveAttempt.EntityID,
			FromX:    oldX,
			FromY:    oldY,
			ToX:      moveAttempt.ToX,
			ToY:      moveAttempt.ToY,
		})
	}
}

// Update handles entity movement
func (s *MovementSystem) Update(world *ecs.World, dt float64) {
	// Reset movement attempt flag each frame
	s.moveAttempted = false
}

// isValidMoveStandard checks if movement is valid on a standard map
func (s *MovementSystem) isValidMoveStandard(world *ecs.World, mapID ecs.EntityID, x, y int, entityID ecs.EntityID) bool {
	// Get map component
	mapComp, exists := world.GetComponent(mapID, components.MapComponentID)
	if !exists {
		return false
	}
	mapData := mapComp.(*components.MapComponent)

	// Check for walls
	if mapData.IsWall(x, y) {
		return false
	}

	// Check for entity collision
	targetID := s.getEntityAtPosition(world, x, y)
	if targetID != 0 && targetID != entityID {
		// If there's an entity and it's not the moving entity
		if _, hasCollision := world.GetComponent(targetID, components.Collision); hasCollision {
			// Emit a collision event
			world.EmitEvent(CollisionEvent{
				EntityID1: entityID,
				EntityID2: targetID,
				X:         x,
				Y:         y,
			})
			return false
		}
	}

	return true
}

// getEntityAtPosition returns an entity ID at the specified position
func (s *MovementSystem) getEntityAtPosition(world *ecs.World, x, y int) ecs.EntityID {
	// Get all entities with position components
	for _, entity := range world.GetAllEntities() {
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		if !hasPos {
			continue
		}

		pos := posComp.(*components.PositionComponent)
		if pos.X == x && pos.Y == y {
			// Found an entity at the target position
			return entity.ID
		}
	}

	return 0 // No entity found (using 0 as invalid ID)
}
