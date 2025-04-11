package systems

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
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

// PlayerTurnProcessorSystem handles all player input and turns
type PlayerTurnProcessorSystem struct {
	// Map of keys to movement directions
	movementKeys map[ebiten.Key]int
	// Time tracking for continuous movement
	moveDelayTimer      float64
	initialMoveDelay    float64 // Delay before continuous movement starts
	continuousMoveDelay float64 // Delay between continuous movements
	lastDirection       int     // Last movement direction
}

// NewPlayerTurnProcessorSystem creates a new player turn processor system
func NewPlayerTurnProcessorSystem() *PlayerTurnProcessorSystem {
	system := &PlayerTurnProcessorSystem{
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

	// Regular number keys (following numpad layout)
	system.movementKeys[ebiten.Key8] = DirUp
	system.movementKeys[ebiten.Key2] = DirDown
	system.movementKeys[ebiten.Key4] = DirLeft
	system.movementKeys[ebiten.Key6] = DirRight
	system.movementKeys[ebiten.Key7] = DirUpLeft
	system.movementKeys[ebiten.Key9] = DirUpRight
	system.movementKeys[ebiten.Key1] = DirDownLeft
	system.movementKeys[ebiten.Key3] = DirDownRight

	return system
}

// Update processes player input and emits appropriate events
func (s *PlayerTurnProcessorSystem) Update(world *ecs.World, dt float64) {
	// Update movement timer
	s.moveDelayTimer -= dt

	// Process player input
	playerActed := s.processPlayerInput(world)

	// If player took an action, set a flag or emit a global event that the turn is complete
	if playerActed {
		// Emit a turn completed event that other systems can react to
		world.EmitEvent(TurnCompletedEvent{
			EntityID: s.getPlayerID(world),
		})
	}
}

// processPlayerInput handles all player input and returns true if the player took an action
func (s *PlayerTurnProcessorSystem) processPlayerInput(world *ecs.World) bool {
	// Get player entity
	playerID := s.getPlayerID(world)
	if playerID == 0 {
		return false
	}

	// Process rest action
	if s.checkRestInput() {
		s.processRestAction(world, playerID)
		return true
	}

	// Process movement input
	direction, moved := s.getMovementDirection()

	// Handle movement cooldown
	if !moved && s.moveDelayTimer > 0 {
		return false
	}

	// If direction changed or movement just started, reset the timer
	if moved {
		s.lastDirection = direction
		s.moveDelayTimer = s.initialMoveDelay
	}

	// Handle continuous movement
	if s.lastDirection != DirNone && s.moveDelayTimer <= 0 {
		direction = s.lastDirection
		s.moveDelayTimer = s.continuousMoveDelay
	} else if !moved {
		// No new key press and not ready for continuous movement
		return false
	}

	// Handle movement in the chosen direction
	if direction != DirNone {
		return s.processMovementAction(world, playerID, direction)
	}

	return false
}

// getPlayerID returns the player entity ID or 0 if not found
func (s *PlayerTurnProcessorSystem) getPlayerID(world *ecs.World) ecs.EntityID {
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return 0
	}
	return playerEntities[0].ID
}

// checkRestInput returns true if the player pressed a rest key
func (s *PlayerTurnProcessorSystem) checkRestInput() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyNumpad5) ||
		inpututil.IsKeyJustPressed(ebiten.Key5) ||
		inpututil.IsKeyJustPressed(ebiten.KeyPeriod)
}

// processRestAction handles the rest action
func (s *PlayerTurnProcessorSystem) processRestAction(world *ecs.World, playerID ecs.EntityID) {
	// Debug message
	GetDebugLog().Add("DEBUG: Rest action triggered")

	// Emit rest event
	world.EmitEvent(RestEvent{
		EntityID: playerID,
	})
	GetMessageLog().Add("You take a moment to rest.")

	// Add debug log for player stats
	statsComp, hasStats := world.GetComponent(playerID, components.Stats)
	if hasStats {
		stats := statsComp.(*components.StatsComponent)
		GetDebugLog().Add("DEBUG: Player health: " + strconv.Itoa(stats.Health) + "/" +
			strconv.Itoa(stats.MaxHealth) + ", HealingFactor: " + strconv.Itoa(stats.HealingFactor))
	}
}

// processMovementAction handles player movement and returns true if movement was attempted
func (s *PlayerTurnProcessorSystem) processMovementAction(world *ecs.World, playerID ecs.EntityID, direction int) bool {
	// Get player position
	posComp, hasPos := world.GetComponent(playerID, components.Position)
	if !hasPos {
		return false
	}
	position := posComp.(*components.PositionComponent)

	// Calculate movement delta
	dx, dy := s.getDeltaFromDirection(direction)

	// Emit player movement attempt event
	world.EmitEvent(PlayerMoveAttemptEvent{
		EntityID:  playerID,
		FromX:     position.X,
		FromY:     position.Y,
		ToX:       position.X + dx,
		ToY:       position.Y + dy,
		Direction: direction,
	})

	return true
}

// getMovementDirection checks for pressed keys and returns the movement direction
func (s *PlayerTurnProcessorSystem) getMovementDirection() (int, bool) {
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
func (s *PlayerTurnProcessorSystem) getDeltaFromDirection(dir int) (int, int) {
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
