package main

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
	"ebiten-rogue/systems"
)

// WorldMapTester implements ebiten.Game interface for testing the world map.
type WorldMapTester struct {
	world        *ecs.World
	renderSystem *systems.RenderSystem
	mapSystem    *systems.MapSystem
	cameraSystem *systems.CameraSystem
	worldMap     *ecs.Entity
	mapComp      *components.MapComponent
}

// NewWorldMapTester creates a new world map tester
func NewWorldMapTester() *WorldMapTester {
	// Initialize ECS world
	world := ecs.NewWorld()

	// Create systems
	tileset, err := systems.NewTileset("Nice_curses_12x12.png", config.TileSize)
	if err != nil {
		panic(err)
	}

	// Initialize systems
	mapSystem := systems.NewMapSystem()
	cameraSystem := systems.NewCameraSystem()
	renderSystem := systems.NewRenderSystem(tileset)

	// Connect the camera system to the render system
	renderSystem.SetCameraSystem(cameraSystem)

	// Register systems with the world that need to be updated during the game loop
	world.AddSystem(mapSystem)
	world.AddSystem(cameraSystem)

	// Create the tester
	tester := &WorldMapTester{
		world:        world,
		renderSystem: renderSystem,
		mapSystem:    mapSystem,
		cameraSystem: cameraSystem,
	}
	// Create a tile mapping entity first (needed by the render system)
	tileMapEntity := tester.world.CreateEntity()
	tileMapEntity.AddTag("tilemap")
	tester.world.TagEntity(tileMapEntity.ID, "tilemap")

	// Add the tile mapping component with default definitions
	tester.world.AddComponent(tileMapEntity.ID, components.Appearance, components.NewTileMappingComponent())

	// Initialize the world map
	tester.initialize()

	return tester
}

// initialize creates the world map for testing
func (g *WorldMapTester) initialize() { // First, generate the world map
	worldMapGenerator := generation.NewWorldMapGenerator(time.Now().UnixNano())
	g.worldMap = worldMapGenerator.CreateWorldMapEntity(g.world, 200, 200)

	// Add the "map" tag to the world map entity - this is critical
	g.worldMap.AddTag("map")
	g.world.TagEntity(g.worldMap.ID, "map")

	// Add map type component to the world map
	g.world.AddComponent(g.worldMap.ID, components.MapType,
		components.NewMapTypeComponent("worldmap", 0))

	// Get the map component
	if comp, exists := g.world.GetComponent(g.worldMap.ID, components.MapComponentID); exists {
		g.mapComp = comp.(*components.MapComponent)
	} else {
		systems.GetMessageLog().Add("Error: Failed to get map component")
		return
	}

	// Set the active map in the map system
	g.mapSystem.SetActiveMap(g.worldMap)
	// Create a camera entity for viewing the world map
	cameraEntity := g.world.CreateEntity()
	g.world.AddComponent(cameraEntity.ID, components.Camera, &components.CameraComponent{
		X:      100, // Center of the world map (200x200)
		Y:      100,
		Target: 0, // No target entity
	})

	// Add instruction message
	systems.GetMessageLog().Add("World Map Tester - Use arrow keys to move the camera")
}

// Update updates the game state
func (g *WorldMapTester) Update() error {
	// Handle camera movement with arrow keys
	moveSpeed := 5

	// Get camera entity
	cameraEntities := g.world.GetEntitiesWithTag("camera")
	if len(cameraEntities) > 0 {
		cameraID := cameraEntities[0].ID
		cameraComp, exists := g.world.GetComponent(cameraID, components.Camera)
		if exists {
			camera := cameraComp.(*components.CameraComponent)

			// Move camera based on arrow key input
			if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
				camera.Y -= moveSpeed
			}
			if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
				camera.Y += moveSpeed
			}
			if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
				camera.X -= moveSpeed
			}
			if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
				camera.X += moveSpeed
			}
		}
	}

	// Update the render system
	g.renderSystem.Update(g.world, 1.0/60.0)
	return nil
}

// Draw draws the game screen
func (g *WorldMapTester) Draw(screen *ebiten.Image) {
	// Use the render system to draw the world map
	g.renderSystem.Draw(g.world, screen)

	// Print FPS and coordinates for debugging
	camX, camY := 0, 0

	// Get camera position from the camera entity
	cameraEntities := g.world.GetEntitiesWithTag("camera")
	if len(cameraEntities) > 0 {
		cameraID := cameraEntities[0].ID
		cameraComp, exists := g.world.GetComponent(cameraID, components.Camera)
		if exists {
			camera := cameraComp.(*components.CameraComponent)
			camX, camY = camera.X, camera.Y
		}
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.1f | Camera: %d,%d",
		ebiten.ActualFPS(), camX, camY))
}

// Layout implements ebiten.Game's Layout
func (g *WorldMapTester) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}
