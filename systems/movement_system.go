package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
)
// Direction constants for movement
const (
	DirNone = iota
	DirUp
	DirDown
	DirLeft
	DirRight
	DirUpLeft
	DirUpRight
	DirDownLeft
	DirDownRight
)

// MovementSystem handles entity movement
type MovementSystem struct {
	// Map of keys to movement directions
	movementKeys map[ebiten.Key]int
	// Time tracking for continuous movement
	moveDelayTimer      float64
	initialMoveDelay    float64 // Delay before continuous movement starts
	continuousMoveDelay float64 // Delay between continuous movements
	lastDirection       int     // Last movement direction
}

// NewMovementSystem creates a new movement system
func NewMovementSystem() *MovementSystem {
	system := &MovementSystem{
		movementKeys:        make(map[ebiten.Key]int),
		initialMoveDelay:    0.25, // Wait 0.25 seconds before continuous movement starts
		continuousMoveDelay: 0.10, // Then move every 0.10 seconds
		moveDelayTimer:      0,
		lastDirection:       DirNone,
	}

	// Set up default key bindings
	// Arrow keys
	system.movementKeys[ebiten.KeyArrowUp] = DirUp
	system.movementKeys[ebiten.KeyArrowDown] = DirDown
	system.movementKeys[ebiten.KeyArrowLeft] = DirLeft
	system.movementKeys[ebiten.KeyArrowRight] = DirRight

	// Vi keys (hjkl)
	system.movementKeys[ebiten.KeyH] = DirLeft
	system.movementKeys[ebiten.KeyJ] = DirDown
	system.movementKeys[ebiten.KeyK] = DirUp
	system.movementKeys[ebiten.KeyL] = DirRight
	system.movementKeys[ebiten.KeyY] = DirUpLeft
	system.movementKeys[ebiten.KeyU] = DirUpRight
	system.movementKeys[ebiten.KeyB] = DirDownLeft
	system.movementKeys[ebiten.KeyN] = DirDownRight

	// Numpad (if Num Lock is on)
	system.movementKeys[ebiten.KeyNumpad8] = DirUp
	system.movementKeys[ebiten.KeyNumpad2] = DirDown
	system.movementKeys[ebiten.KeyNumpad4] = DirLeft
	system.movementKeys[ebiten.KeyNumpad6] = DirRight
	system.movementKeys[ebiten.KeyNumpad7] = DirUpLeft
	system.movementKeys[ebiten.KeyNumpad9] = DirUpRight
	system.movementKeys[ebiten.KeyNumpad1] = DirDownLeft
	system.movementKeys[ebiten.KeyNumpad3] = DirDownRight

	return system
}

// Update handles entity movement
func (s *MovementSystem) Update(world *ecs.World, dt float64) {
	// Update movement timer
	s.moveDelayTimer -= dt

	// Process player movement
	s.processPlayerMovement(world)

	// No need to directly access the AI system - it will listen for player movement events
}

// processPlayerMovement handles player input and movement
// Returns true if the player moved
func (s *MovementSystem) processPlayerMovement(world *ecs.World) bool {
	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return false
	}

	playerID := playerEntities[0].ID

	// Check for movement input
	dir, moved := s.getMovementDirection()

	// If no movement or timer is not ready, return
	if !moved && s.moveDelayTimer > 0 {
		return false
	}

	// If direction changed or movement just started, reset the timer with initial delay
	if moved {
		s.lastDirection = dir
		s.moveDelayTimer = s.initialMoveDelay
	}

	// If a key is being held and timer is ready, process movement with continuous delay
	if s.lastDirection != DirNone && s.moveDelayTimer <= 0 {
		dir = s.lastDirection
		s.moveDelayTimer = s.continuousMoveDelay
	} else if !moved {
		// No new key press and not ready for continuous movement
		return false
	}

	// Get player position
	var position *components.PositionComponent
	if comp, exists := world.GetComponent(playerID, components.Position); exists {
		position = comp.(*components.PositionComponent)
	} else {
		return false
	}

	// Calculate movement delta
	dx, dy := s.getDeltaFromDirection(dir)

	// Store old position
	oldX, oldY := position.X, position.Y

	// Calculate new position
	newX := position.X + dx
	newY := position.Y + dy

	// Check if we're using a standard map
	standardMapEntities := world.GetEntitiesWithTag("map")

	// Check if the move is valid
	canMove := false

	if len(standardMapEntities) > 0 {
		// Using standard map
		canMove = s.isValidMoveStandard(world, standardMapEntities[0].ID, newX, newY, playerID)
	} else {
		// No map found, assume can move
		canMove = true
	}

	// If move is valid, update position
	if canMove {
		position.X = newX
		position.Y = newY

		// Emit movement event
		world.EmitEvent(PlayerMoveEvent{
			EntityID: playerID,
			FromX:    oldX,
			FromY:    oldY,
			ToX:      newX,
			ToY:      newY,
		})
		// Log the movement
		GetMessageLog().Add(fmt.Sprintf("Moved to (%d, %d)", newX, newY))

		return true // The player moved
	} else {
		// If we can't move, clear the last direction to prevent continuous movement into walls
		s.lastDirection = DirNone
		return false // The player didn't move
	}
}

// isValidMoveStandard checks if movement is valid on a standard map
func (s *MovementSystem) isValidMoveStandard(world *ecs.World, mapID ecs.EntityID, x, y int, playerID ecs.EntityID) bool {
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
	if targetID != 0 && targetID != playerID {
		// If there's an entity and it's not the player
		if _, hasCollision := world.GetComponent(targetID, components.Collision); hasCollision {
			// Emit a collision event
			world.EmitEvent(CollisionEvent{
				EntityID1: playerID,
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

// isPositionBlocked checks if a position is blocked by any entity with a collision component
func (s *MovementSystem) isPositionBlocked(world *ecs.World, x, y int) bool {
	entityID := s.getEntityAtPosition(world, x, y)
	if entityID == 0 {
		return false
	}

	// Check if the entity has a collision component that blocks movement
	collComp, hasCollision := world.GetComponent(entityID, components.Collision)
	if !hasCollision {
		return false
	}

	collision := collComp.(*components.CollisionComponent)
	return collision.Blocks
}

// getMovementDirection checks for pressed keys and returns the movement direction
func (s *MovementSystem) getMovementDirection() (int, bool) {
	// First check for newly pressed keys - these take priority
	for key, dir := range s.movementKeys {
		if inpututil.IsKeyJustPressed(key) {
			return dir, true
		}
	}

	// Then check for held keys - this is what enables continuous movement
	for key, dir := range s.movementKeys {
		if ebiten.IsKeyPressed(key) {
			// If any key is currently pressed, check if it's a new direction
			if dir != s.lastDirection {
				return dir, true
			}
			// If it's the same direction as before, just notify that a key is being held
			if dir == s.lastDirection {
				return DirNone, false
			}
		}
	}

	// No movement key is pressed, reset the last direction
	s.lastDirection = DirNone
	return DirNone, false
}

// getDeltaFromDirection converts a direction to dx, dy coordinates
func (s *MovementSystem) getDeltaFromDirection(dir int) (int, int) {
	dx, dy := 0, 0

	switch dir {
	case DirUp:
		dy = -1
	case DirDown:
		dy = 1
	case DirLeft:
		dx = -1
	case DirRight:
		dx = 1
	case DirUpLeft:
		dx, dy = -1, -1
	case DirUpRight:
		dx, dy = 1, -1
	case DirDownLeft:
		dx, dy = -1, 1
	case DirDownRight:
		dx, dy = 1, 1
	}

	return dx, dy
}
