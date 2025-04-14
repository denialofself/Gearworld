package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
	"ebiten-rogue/systems"
	"image/color"
)

// WorldMapTester implements ebiten.Game interface for testing the world map.
type WorldMapTester struct {
	world             *ecs.World
	renderSystem      *systems.RenderSystem
	mapSystem         *systems.MapSystem
	cameraSystem      *systems.CameraSystem
	mapRegistrySystem *systems.MapRegistrySystem
	worldMap          *ecs.Entity
	mapComp           *components.MapComponent
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
	mapRegistrySystem := systems.NewMapRegistrySystem()

	// Initialize systems that need the world reference
	mapRegistrySystem.Initialize(world)

	// Register systems with the world that need to be updated during the game loop
	world.AddSystem(mapSystem)
	world.AddSystem(cameraSystem)
	world.AddSystem(renderSystem)
	world.AddSystem(mapRegistrySystem)

	// Create the tester
	tester := &WorldMapTester{
		world:             world,
		renderSystem:      renderSystem,
		mapSystem:         mapSystem,
		cameraSystem:      cameraSystem,
		mapRegistrySystem: mapRegistrySystem,
	}

	// Create a tile mapping entity first (needed by the render system)
	tileMapEntity := tester.world.CreateEntity()
	tileMapEntity.AddTag("tilemap")
	tester.world.TagEntity(tileMapEntity.ID, "tilemap")

	// Add the tile mapping component with default definitions
	tester.world.AddComponent(tileMapEntity.ID, components.Appearance, components.NewTileMappingComponent())

	// Initialize the world map
	tester.initialize()

	// Initialize the render system
	renderSystem.Initialize(world)

	return tester
}

// initialize creates the world map for testing
func (g *WorldMapTester) initialize() {
	// First, generate the world map
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

	// Set the active map in both map system and map registry
	g.mapSystem.SetActiveMap(g.worldMap)
	g.mapRegistrySystem.RegisterMap(g.worldMap)
	g.mapRegistrySystem.SetActiveMap(g.worldMap)

	// Create a camera entity for viewing the world map
	cameraEntity := g.world.CreateEntity()
	cameraEntity.AddTag("camera") // Add camera tag
	g.world.TagEntity(cameraEntity.ID, "camera")
	g.world.AddComponent(cameraEntity.ID, components.Camera, &components.CameraComponent{
		X:      100, // Center of the world map (200x200)
		Y:      100,
		Target: 0, // No target entity
	})

	// Add instruction message
	systems.GetMessageLog().Add("World Map Tester - Use arrow keys to move the camera")
	systems.GetMessageLog().Add("Press F to toggle fullscreen")
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

			// Keep camera within map bounds
			if camera.X < 0 {
				camera.X = 0
			}
			if camera.X >= g.mapComp.Width {
				camera.X = g.mapComp.Width - 1
			}
			if camera.Y < 0 {
				camera.Y = 0
			}
			if camera.Y >= g.mapComp.Height {
				camera.Y = g.mapComp.Height - 1
			}

			// Handle fullscreen toggle
			if inpututil.IsKeyJustPressed(ebiten.KeyF) {
				ebiten.SetFullscreen(!ebiten.IsFullscreen())
			}
		}
	}

	// Update the render system
	g.renderSystem.Update(g.world, 1.0/60.0)
	return nil
}

// Draw draws the game screen
func (g *WorldMapTester) Draw(screen *ebiten.Image) {
	// Clear the screen with black background
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Use the render system to draw the world map
	g.renderSystem.Draw(g.world, screen)
}

// Layout implements ebiten.Game's Layout
func (g *WorldMapTester) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Use the full screen size
	return outsideWidth, outsideHeight
}
