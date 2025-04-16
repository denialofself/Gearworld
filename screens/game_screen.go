package screens

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
	"ebiten-rogue/systems"
)

// GameScreen handles the main gameplay state
type GameScreen struct {
	*BaseScreen
	world                     *ecs.World
	renderSystem              *systems.RenderSystem
	mapSystem                 *systems.MapSystem
	mapRegistrySystem         *systems.MapRegistrySystem
	movementSystem            *systems.MovementSystem
	playerTurnProcessorSystem *systems.PlayerTurnProcessorSystem
	combatSystem              *systems.CombatSystem
	cameraSystem              *systems.CameraSystem
	aiPathfindingSystem       *systems.AIPathfindingSystem
	aiTurnProcessorSystem     *systems.AITurnProcessorSystem
	effectsSystem             *systems.EffectsSystem
	inventorySystem           *systems.InventorySystem
	equipmentSystem           *systems.EquipmentSystem
	fovSystem                 *systems.FOVSystem
	containerSystem           *systems.ContainerSystem
	audioSystem               *systems.AudioSystem
	deathSystem               *systems.DeathSystem
	cachedScreen              *ebiten.Image
	needsRedraw               bool
	screenStack               *ScreenStack
}

// NewGameScreen creates a new game screen
func NewGameScreen(
	world *ecs.World,
	renderSystem *systems.RenderSystem,
	mapSystem *systems.MapSystem,
	mapRegistrySystem *systems.MapRegistrySystem,
	movementSystem *systems.MovementSystem,
	playerTurnProcessorSystem *systems.PlayerTurnProcessorSystem,
	combatSystem *systems.CombatSystem,
	cameraSystem *systems.CameraSystem,
	aiPathfindingSystem *systems.AIPathfindingSystem,
	aiTurnProcessorSystem *systems.AITurnProcessorSystem,
	effectsSystem *systems.EffectsSystem,
	inventorySystem *systems.InventorySystem,
	equipmentSystem *systems.EquipmentSystem,
	fovSystem *systems.FOVSystem,
	containerSystem *systems.ContainerSystem,
	audioSystem *systems.AudioSystem,
	deathSystem *systems.DeathSystem,
) *GameScreen {
	return &GameScreen{
		BaseScreen:                NewBaseScreen(),
		world:                     world,
		renderSystem:              renderSystem,
		mapSystem:                 mapSystem,
		mapRegistrySystem:         mapRegistrySystem,
		movementSystem:            movementSystem,
		playerTurnProcessorSystem: playerTurnProcessorSystem,
		combatSystem:              combatSystem,
		cameraSystem:              cameraSystem,
		aiPathfindingSystem:       aiPathfindingSystem,
		aiTurnProcessorSystem:     aiTurnProcessorSystem,
		effectsSystem:             effectsSystem,
		inventorySystem:           inventorySystem,
		equipmentSystem:           equipmentSystem,
		fovSystem:                 fovSystem,
		containerSystem:           containerSystem,
		audioSystem:               audioSystem,
		deathSystem:               deathSystem,
		needsRedraw:               true,
		screenStack:               NewScreenStack(),
	}
}

// Update handles game updates
func (s *GameScreen) Update() error {
	// Toggle debug message window with F1 key
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		if s.screenStack.Peek() != nil {
			// If there's a screen on the stack, pop it (close debug screen)
			s.screenStack.Pop()
		} else {
			// Otherwise, push the debug screen onto the stack
			debugScreen := NewDebugScreen()
			s.screenStack.Push(debugScreen)
		}
		s.needsRedraw = true
	}

	// Don't process input if a map transition is in progress
	if s.mapRegistrySystem.IsTransitionInProgress() {
		systems.GetDebugLog().Add("Update skipped: map transition in progress")
		return nil
	}

	// Update the screen stack first to handle modal input
	if err := s.screenStack.Update(); err != nil {
		if err == ErrCloseScreen {
			s.screenStack.Pop()
		}
		s.needsRedraw = true
	}

	// Only update the game world if no modal is open
	if s.screenStack.Peek() == nil {
		// Update all systems - player input will be handled by PlayerTurnProcessorSystem
		s.world.Update(1.0 / 60.0)
		// Always redraw after updating systems
		s.needsRedraw = true
	}

	return nil
}

// Draw draws the game screen
func (s *GameScreen) Draw(screen *ebiten.Image) {
	// Draw the game world
	s.renderSystem.Draw(s.world, screen)

	// If there's a screen on the stack, draw it
	if s.screenStack.Peek() != nil {
		s.screenStack.Draw(screen)
	}
}

// Layout implements the Screen interface
func (s *GameScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}
