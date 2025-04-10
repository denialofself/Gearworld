package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
	"ebiten-rogue/spawners"
	"ebiten-rogue/systems"
)

// Game implements ebiten.Game interface.
type Game struct {
	world                 *ecs.World
	renderSystem          *systems.RenderSystem
	mapSystem             *systems.MapSystem
	movementSystem        *systems.MovementSystem
	combatSystem          *systems.CombatSystem
	cameraSystem          *systems.CameraSystem
	templateManager       *data.EntityTemplateManager
	entitySpawner         *spawners.EntitySpawner
	aiPathfindingSystem   *systems.AIPathfindingSystem
	aiTurnProcessorSystem *systems.AITurnProcessorSystem
}

// NewGame creates a new game instance
func NewGame() *Game {
	// Initialize ECS world
	world := ecs.NewWorld()

	// Create systems
	tileset, err := systems.NewTileset("Nice_curses_12x12.png", config.TileSize)
	if err != nil {
		panic(err)
	}

	// Initialize all systems
	mapSystem := systems.NewMapSystem()
	movementSystem := systems.NewMovementSystem()
	combatSystem := systems.NewCombatSystem()
	cameraSystem := systems.NewCameraSystem()
	renderSystem := systems.NewRenderSystem(tileset)
	aiPathfindingSystem := systems.NewAIPathfindingSystem()
	aiTurnProcessorSystem := systems.NewAITurnProcessorSystem()

	// Initialize the entity template manager
	templateManager := data.NewEntityTemplateManager()

	// Load monster templates
	err = templateManager.LoadTemplatesFromDirectory("data/monsters")
	if err != nil {
		fmt.Printf("Warning: Failed to load monster templates: %v\n", err)
	}

	// Create entity spawner
	entitySpawner := spawners.NewEntitySpawner(world, templateManager, systems.GetMessageLog().Add)

	// Connect the camera system to the render system
	renderSystem.SetCameraSystem(cameraSystem)

	// Register systems with the world that need to be updated during the game loop
	world.AddSystem(movementSystem)
	world.AddSystem(combatSystem)
	world.AddSystem(cameraSystem)
	world.AddSystem(mapSystem)
	world.AddSystem(aiPathfindingSystem)
	world.AddSystem(aiTurnProcessorSystem)

	// Create the game instance
	game := &Game{
		world:                 world,
		renderSystem:          renderSystem,
		mapSystem:             mapSystem,
		movementSystem:        movementSystem,
		combatSystem:          combatSystem,
		cameraSystem:          cameraSystem,
		templateManager:       templateManager,
		entitySpawner:         entitySpawner,
		aiPathfindingSystem:   aiPathfindingSystem,
		aiTurnProcessorSystem: aiTurnProcessorSystem,
	}

	// Initialize the game world
	game.initialize()

	// Initialize event listeners
	combatSystem.Initialize(world)
	aiPathfindingSystem.Initialize(world)
	aiTurnProcessorSystem.Initialize(world)

	return game
}

// initialize sets up the initial game state
func (g *Game) initialize() {
	// Create the tile mapping entity
	g.entitySpawner.CreateTileMapping()

	// Create a dungeon themer
	dungeonThemer := generation.NewDungeonThemer(
		g.world,
		g.templateManager,
		g.entitySpawner,
		systems.GetMessageLog().Add, // Pass the logging function
	)

	// Set a random seed for dungeon generation
	dungeonThemer.SetSeed(time.Now().UnixNano())

	// Configure the dungeon (level 1, abandoned theme, large size)
	config := generation.DungeonConfiguration{
		Level:                 1,
		Theme:                 generation.ThemeAbandoned,
		Size:                  generation.SizeLarge,
		Generator:             generation.GeneratorBSP,
		DensityFactor:         .30,
		HigherLevelChance:     0.05,  // 5% chance for level 2 monsters
		EvenHigherLevelChance: 0.01,  // 1% chance for level 3 monsters
		AddStairsUp:           false, // No need for stairs up since we're removing world map
	}

	// Generate the themed dungeon with appropriate monsters
	dungeonEntity := dungeonThemer.GenerateThemedDungeon(config)

	// Get the map component from the dungeon entity
	var mapComp *components.MapComponent
	if comp, exists := g.world.GetComponent(dungeonEntity.ID, components.MapComponentID); exists {
		mapComp = comp.(*components.MapComponent)
	}

	if mapComp == nil {
		systems.GetMessageLog().Add("Error: Failed to get map component")
		return
	}

	// Set the active map in the map system
	g.mapSystem.SetActiveMap(dungeonEntity)

	// Find empty position for player
	playerX, playerY := g.mapSystem.FindEmptyPosition(mapComp)

	// Create the player entity
	playerEntity := g.entitySpawner.CreatePlayer(playerX, playerY)

	// Create a camera entity for the player
	g.entitySpawner.CreateCamera(uint64(playerEntity.ID), playerX, playerY)

	// Add welcome message
	systems.GetMessageLog().Add("Welcome to the abandoned dungeon!")

	// Add instruction message
	systems.GetMessageLog().Add("Use arrow keys to move.")
}

// Update updates the game state.
func (g *Game) Update() error { // Toggle debug message window with F1 key
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.renderSystem.ToggleDebugWindow()
	}
	// If debug window is active
	if g.renderSystem.IsDebugWindowActive() {
		// ESC to close debug window
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			g.renderSystem.ToggleDebugWindow()
		}

		// Handle scrolling through debug messages with arrow keys
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.renderSystem.ScrollDebugUp()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.renderSystem.ScrollDebugDown()
		}

		// Only update the render system when debug window is active
		g.renderSystem.Update(g.world, 1.0/60.0)
		return nil
	}

	// Check if the user wants to view the tileset
	if ebiten.IsKeyPressed(ebiten.KeyF12) {
		go func() {
			cmd := exec.Command(os.Args[0], "--view-tileset")
			err := cmd.Start()
			if err != nil {
				systems.GetMessageLog().Add("Error launching tileset viewer: " + err.Error())
			}
		}()
	}

	// Update all systems
	g.world.Update(1.0 / 60.0) // passing approximate dt value

	// Update render system separately (not part of world systems)
	g.renderSystem.Update(g.world, 1.0/60.0)

	return nil
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Use the render system to draw the game
	g.renderSystem.Draw(g.world, screen)

	// Print FPS for debugging
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.1f", ebiten.ActualFPS()))
}

// Layout implements ebiten.Game's Layout.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}
