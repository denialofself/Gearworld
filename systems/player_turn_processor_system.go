package systems

import (
	"fmt"
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

	// Reference to the render system for UI state changes
	renderSystem *RenderSystem
}

// NewPlayerTurnProcessorSystem creates a new player turn processor system
func NewPlayerTurnProcessorSystem() *PlayerTurnProcessorSystem {
	system := &PlayerTurnProcessorSystem{
		movementKeys:        make(map[ebiten.Key]int),
		initialMoveDelay:    0.25, // Wait 0.25 seconds before continuous movement starts
		continuousMoveDelay: 0.10, // Then move every 0.10 seconds
		moveDelayTimer:      0,
		lastDirection:       DirNone,
		renderSystem:        nil,
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

// SetRenderSystem sets the reference to the render system for UI state changes
func (s *PlayerTurnProcessorSystem) SetRenderSystem(renderSystem *RenderSystem) {
	s.renderSystem = renderSystem
}

// Update processes player input and emits appropriate events
func (s *PlayerTurnProcessorSystem) Update(world *ecs.World, dt float64) {
	// Find render system if not set
	if s.renderSystem == nil {
		for _, system := range world.GetSystems() {
			if renderSys, ok := system.(*RenderSystem); ok {
				s.renderSystem = renderSys
				break
			}
		}
	}

	// Update movement timer
	s.moveDelayTimer -= dt

	// Check for inventory toggle first, which doesn't count as a turn
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		s.toggleInventory()
		return
	}

	// If inventory is open, process inventory-specific inputs
	if s.renderSystem != nil && s.renderSystem.IsInventoryOpen() {
		s.processInventoryInput(world)
		return
	}

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

// toggleInventory toggles the inventory display
func (s *PlayerTurnProcessorSystem) toggleInventory() {
	if s.renderSystem != nil {
		s.renderSystem.ToggleInventoryDisplay()
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

// processInventoryInput handles keyboard input while the inventory is open
func (s *PlayerTurnProcessorSystem) processInventoryInput(world *ecs.World) {
	// Check for ESC to close inventory or exit item view mode
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.renderSystem.IsItemViewMode() {
			s.renderSystem.ExitItemView()
		} else {
			s.renderSystem.ToggleInventoryDisplay()
		}
		return
	}

	// Get player entity
	playerID := s.getPlayerID(world)
	if playerID == 0 {
		return
	}

	// Check if player has an inventory
	var inventory *components.InventoryComponent
	if comp, exists := world.GetComponent(playerID, components.Inventory); exists {
		inventory = comp.(*components.InventoryComponent)
	} else {
		return // No inventory, nothing to process
	}

	// Process 'L' key for looking at items
	if inpututil.IsKeyJustPressed(ebiten.KeyL) {
		if s.renderSystem.IsItemViewMode() {
			// Exit item view mode if already in it
			s.renderSystem.ExitItemView()
			GetMessageLog().Add("Returned to inventory view")
		} else {
			// Find which item is selected
			// For now, we'll just use the first item in the inventory
			// In a more advanced implementation, you might want to track the currently selected item
			if inventory.Size() > 0 {
				s.renderSystem.ViewItemDetails(0)
				// Get item name if possible
				itemID := inventory.Items[0]
				itemName := "item"
				if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
					itemName = nameComp.(*components.NameComponent).Name
				}
				GetMessageLog().Add(fmt.Sprintf("Examining %s", itemName))
			}
		}
		return
	}

	// Process item selection (keys a-z for items 0-25)
	for i := 0; i < 26 && i < inventory.Size(); i++ {
		// Calculate the correct key code
		key := ebiten.Key(int(ebiten.KeyA) + i)
		if inpututil.IsKeyJustPressed(key) {
			// If we're in item view mode, view that specific item
			if s.renderSystem.IsItemViewMode() {
				s.renderSystem.ViewItemDetails(i)

				// Get item name if possible
				itemID := inventory.Items[i]
				itemName := "item"
				if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
					itemName = nameComp.(*components.NameComponent).Name
				}
				GetMessageLog().Add(fmt.Sprintf("Examining %s", itemName))
			} else {
				// Otherwise, either view it or use it
				// For now, just view it
				s.renderSystem.ViewItemDetails(i)

				// Get item name if possible
				itemID := inventory.Items[i]
				itemName := "item"
				if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
					itemName = nameComp.(*components.NameComponent).Name
				}
				GetMessageLog().Add(fmt.Sprintf("Examining %s", itemName))
			}
			return
		}
	}
}
