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
	world                     *ecs.World
	renderSystem              *systems.RenderSystem
	mapSystem                 *systems.MapSystem
	mapRegistrySystem         *systems.MapRegistrySystem
	movementSystem            *systems.MovementSystem
	playerTurnProcessorSystem *systems.PlayerTurnProcessorSystem
	combatSystem              *systems.CombatSystem
	cameraSystem              *systems.CameraSystem
	templateManager           *data.EntityTemplateManager
	entitySpawner             *spawners.EntitySpawner
	aiPathfindingSystem       *systems.AIPathfindingSystem
	aiTurnProcessorSystem     *systems.AITurnProcessorSystem
	effectsSystem             *systems.EffectsSystem
	inventorySystem           *systems.InventorySystem
	fovSystem                 *systems.FOVSystem
	// Equipment functionality is now handled by the inventory system
}

// NewGame creates a new game instance
func NewGame() *Game {
	// Initialize ECS world
	world := ecs.NewWorld()

	// Create systems
	tileset, err := systems.NewTileset("Nice_curses_12x12.png", config.TileSize)
	if err != nil {
		panic(err)
	} // Initialize all systems
	mapSystem := systems.NewMapSystem()
	mapRegistrySystem := systems.NewMapRegistrySystem()
	movementSystem := systems.NewMovementSystem()
	playerTurnProcessorSystem := systems.NewPlayerTurnProcessorSystem()
	combatSystem := systems.NewCombatSystem()
	cameraSystem := systems.NewCameraSystem()
	renderSystem := systems.NewRenderSystem(tileset)
	aiPathfindingSystem := systems.NewAIPathfindingSystem()
	aiTurnProcessorSystem := systems.NewAITurnProcessorSystem()
	effectsSystem := systems.NewEffectsSystem()
	inventorySystem := systems.NewInventorySystem()
	fovSystem := systems.NewFOVSystem()
	// Equipment functionality is now handled by the inventory system

	// Initialize the entity template manager
	templateManager := data.NewEntityTemplateManager()

	// Load monster templates
	err = templateManager.LoadTemplatesFromDirectory("data/monsters")
	if err != nil {
		fmt.Printf("Warning: Failed to load monster templates: %v\n", err)
	}

	// Load item templates
	err = templateManager.LoadItemTemplatesFromDirectory("data/items")
	if err != nil {
		fmt.Printf("Warning: Failed to load item templates: %v\n", err)
	}

	// Create entity spawner
	entitySpawner := spawners.NewEntitySpawner(world, templateManager, systems.GetMessageLog().Add)

	// Connect the camera system to the render system
	renderSystem.SetCameraSystem(cameraSystem)
	playerTurnProcessorSystem.SetRenderSystem(renderSystem)

	// Register systems with the world that need to be updated during the game loop
	// Register systems with the world
	world.AddSystem(mapSystem)
	world.AddSystem(mapRegistrySystem)
	world.AddSystem(movementSystem)
	world.AddSystem(playerTurnProcessorSystem)
	world.AddSystem(combatSystem)
	world.AddSystem(cameraSystem)
	world.AddSystem(aiPathfindingSystem)
	world.AddSystem(aiTurnProcessorSystem)
	world.AddSystem(effectsSystem)
	world.AddSystem(inventorySystem)
	world.AddSystem(fovSystem)
	// Equipment functionality is now handled by the inventory system

	// Create the game instance
	game := &Game{
		world:                     world,
		renderSystem:              renderSystem,
		mapSystem:                 mapSystem,
		mapRegistrySystem:         mapRegistrySystem,
		movementSystem:            movementSystem,
		playerTurnProcessorSystem: playerTurnProcessorSystem,
		combatSystem:              combatSystem,
		cameraSystem:              cameraSystem,
		templateManager:           templateManager,
		entitySpawner:             entitySpawner,
		aiPathfindingSystem:       aiPathfindingSystem,
		aiTurnProcessorSystem:     aiTurnProcessorSystem,
		effectsSystem:             effectsSystem,
		inventorySystem:           inventorySystem,
		fovSystem:                 fovSystem,
		// Equipment functionality is now handled by the inventory system
	}

	// Initialize the game world
	game.initialize()
	// Initialize event listeners
	movementSystem.Initialize(world)
	combatSystem.Initialize(world)
	aiPathfindingSystem.Initialize(world)
	aiTurnProcessorSystem.Initialize(world)
	effectsSystem.Initialize(world)
	inventorySystem.Initialize(world)
	fovSystem.Initialize(world)
	// Equipment system has been removed - functionality moved to inventory system

	// Call the map debug function
	components.DebugWallDetection()

	systems.GetDebugLog().Add("Game initialization complete")

	return game
}

// initialize sets up the initial game state
func (g *Game) initialize() {
	// Create the tile mapping entity
	g.entitySpawner.CreateTileMapping()

	// Initialize the map registry system
	g.mapRegistrySystem.Initialize(g.world)

	// First, generate a world map
	worldMapGenerator := generation.NewWorldMapGenerator(time.Now().UnixNano())
	worldMapEntity := worldMapGenerator.CreateWorldMapEntity(g.world, 200, 200)

	// Make sure the world map is properly tagged
	worldMapEntity.AddTag("map")
	worldMapEntity.AddTag("worldmap")
	g.world.TagEntity(worldMapEntity.ID, "map")
	g.world.TagEntity(worldMapEntity.ID, "worldmap")

	// Add map type component to the world map
	g.world.AddComponent(worldMapEntity.ID, components.MapType,
		components.NewMapTypeComponent("worldmap", 0))

	// Log the world map entity ID for debugging
	systems.GetMessageLog().Add(fmt.Sprintf("DEBUG: Created world map with ID: %d", worldMapEntity.ID))

	// Register the world map with the map registry
	g.mapRegistrySystem.RegisterMap(worldMapEntity)

	// Create a dungeon themer
	dungeonThemer := generation.NewDungeonThemer(
		g.world,
		g.templateManager,
		g.entitySpawner,
		systems.GetMessageLog().Add, // Pass the logging function
	)

	// Set a random seed for dungeon generation
	dungeonThemer.SetSeed(time.Now().UnixNano())

	// Load dungeon themes from JSON files
	err := dungeonThemer.LoadThemesFromDirectory("data/themes")
	if err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("WARNING: Failed to load dungeon themes: %v", err))
	} else {
		systems.GetMessageLog().Add("Successfully loaded dungeon themes from data/themes")
	}

	// Configure the dungeon (level 1, abandoned theme, large size)
	config := generation.DungeonConfiguration{
		Level:       1,
		Size:        generation.SizeSmall,
		Generator:   generation.GeneratorBSP,
		AddStairsUp: true,               // Add stairs up to return to the world map
		ThemeID:     "starting_station", // Use the JSON theme if available
	}

	// Generate the themed dungeon with appropriate monsters
	startingStationEntity := dungeonThemer.GenerateThemedDungeon(config)

	// Add map type component if it doesn't exist
	if !g.world.HasComponent(startingStationEntity.ID, components.MapType) {
		g.world.AddComponent(startingStationEntity.ID, components.MapType,
			components.NewMapTypeComponent("starting_station", 1))
	}

	// Log the dungeon entity ID for debugging
	systems.GetMessageLog().Add(fmt.Sprintf("DEBUG: Created dungeon with ID: %d", startingStationEntity.ID))

	// Register the dungeon with the map registry
	g.mapRegistrySystem.RegisterMap(startingStationEntity)

	// Get the map component from the dungeon entity
	var mapComp *components.MapComponent
	if comp, exists := g.world.GetComponent(startingStationEntity.ID, components.MapComponentID); exists {
		mapComp = comp.(*components.MapComponent)
	}

	if mapComp == nil {
		systems.GetMessageLog().Add("Error: Failed to get map component")
		return
	}

	// We'll start in the dungeon
	// Set the active map in the map registry system
	g.mapRegistrySystem.SetActiveMap(startingStationEntity)

	// Find empty position for player
	playerX, playerY := g.mapSystem.FindEmptyPosition(mapComp)

	// Create the player entity
	playerEntity := g.entitySpawner.CreatePlayer(playerX, playerY)

	// Add map context component to the player
	g.world.AddComponent(playerEntity.ID, components.MapContextID,
		components.NewMapContextComponent(startingStationEntity.ID))

	// Create test items near the player
	testItemX, testItemY := playerX+2, playerY
	// Create items using templates
	if _, err := g.entitySpawner.CreateItem(testItemX, testItemY, "rusty_spanner"); err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("Failed to create rusty spanner: %v", err))
	}
	if _, err := g.entitySpawner.CreateItem(testItemX+1, testItemY, "bandage"); err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("Failed to create bandage: %v", err))
	}
	// Add our new equipment items
	if _, err := g.entitySpawner.CreateItem(testItemX+1, testItemY+1, "miners_headlamp"); err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("Failed to create miner's headlamp: %v", err))
	}
	if _, err := g.entitySpawner.CreateItem(testItemX+2, testItemY+1, "tattered_jumpsuit"); err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("Failed to create jumpsuit: %v", err))
	}

	// Create a camera entity for the player
	g.entitySpawner.CreateCamera(uint64(playerEntity.ID), playerX, playerY)

	// Print a summary of all maps and their IDs
	g.printMapSummary()

	// Add welcome message
	systems.GetMessageLog().Add("Welcome to the dungeon! Use arrow keys to move.")
}

// Flag to track if we need to redraw the screen
var needsRedraw = true

// Update updates the game state.
func (g *Game) Update() error {
	// Toggle debug message window with F1 key
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.renderSystem.ToggleDebugWindow()
		needsRedraw = true
	}

	// If debug window is active
	if g.renderSystem.IsDebugWindowActive() {
		// ESC to close debug window
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			g.renderSystem.ToggleDebugWindow()
			needsRedraw = true
		}

		// Handle scrolling through debug messages with arrow keys
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.renderSystem.ScrollDebugUp()
			needsRedraw = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.renderSystem.ScrollDebugDown()
			needsRedraw = true
		}

		return nil
	}

	// Check if the user wants to view the tileset
	if ebiten.IsKeyPressed(ebiten.KeyF12) {
		go func() {
			cmd := exec.Command(os.Args[0], "--view-tileset")
			err := cmd.Start()
			if err != nil {
				systems.GetMessageLog().Add("Error launching tileset viewer: " + err.Error())
				needsRedraw = true
			}
		}()
	}

	// Don't process input if a map transition is in progress
	if g.mapRegistrySystem.IsTransitionInProgress() {
		systems.GetDebugLog().Add("Update skipped: map transition in progress")
		return nil
	}

	// Update all systems - player input will be handled by PlayerTurnProcessorSystem
	g.world.Update(1.0 / 60.0)

	// Always redraw after updating systems
	needsRedraw = true

	return nil
}

// Store the current rendered screen to avoid redrawing every frame
var cachedScreen *ebiten.Image

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	// Only redraw if necessary
	if needsRedraw || cachedScreen == nil {
		// Create a new buffer image if we don't have one yet
		if cachedScreen == nil {
			cachedScreen = ebiten.NewImage(config.ScreenWidth*config.TileSize, config.ScreenHeight*config.TileSize)
		}

		// Clear the cached screen before drawing
		cachedScreen.Clear()

		// Use the render system to draw to our cached image
		g.renderSystem.Draw(g.world, cachedScreen)

		// Add FPS counter to the cached image
		ebitenutil.DebugPrintAt(cachedScreen, fmt.Sprintf("FPS: %.1f", ebiten.ActualFPS()), 0, 0)

		// Reset the flag since we've redrawn
		needsRedraw = false
	}

	// Draw the cached screen to the actual screen
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(cachedScreen, op)
}

// Layout implements ebiten.Game's Layout.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}

// printMapDebugInfo outputs debug information about the current map and player position
func (g *Game) printMapDebugInfo() {
	// Get the active map
	activeMap := g.mapRegistrySystem.GetActiveMap()
	if activeMap == nil {
		systems.GetDebugLog().Add("ERROR: No active map found")
		return
	}

	// Get player entity and position
	playerEntities := g.world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		systems.GetDebugLog().Add("ERROR: No player entity found")
		return
	}

	playerEntity := playerEntities[0]
	var playerX, playerY int
	if posComp, exists := g.world.GetComponent(playerEntity.ID, components.Position); exists {
		pos := posComp.(*components.PositionComponent)
		playerX = pos.X
		playerY = pos.Y
	}

	// Get player's map context
	var playerMapID ecs.EntityID
	if contextComp, exists := g.world.GetComponent(playerEntity.ID, components.MapContextID); exists {
		mapContext := contextComp.(*components.MapContextComponent)
		playerMapID = mapContext.MapID

		// Just log if there's a mismatch, but don't auto-correct
		if playerMapID != activeMap.ID {
			systems.GetDebugLog().Add(fmt.Sprintf("MAP MISMATCH: Player is on map %d but active map is %d",
				playerMapID, activeMap.ID))
		}
	}

	// Get map type info
	var mapType string
	var mapLevel int
	if typeComp, exists := g.world.GetComponent(activeMap.ID, components.MapType); exists {
		mapTypeComp := typeComp.(*components.MapTypeComponent)
		mapType = mapTypeComp.MapType
		mapLevel = mapTypeComp.Level
	} else {
		mapType = "unknown"
		mapLevel = -1
	}

	// Get tile info at player position
	tileType := -1
	if mapComp, exists := g.world.GetComponent(activeMap.ID, components.MapComponentID); exists {
		mapData := mapComp.(*components.MapComponent)
		if playerX >= 0 && playerX < mapData.Width && playerY >= 0 && playerY < mapData.Height {
			tileType = mapData.Tiles[playerY][playerX]
		}
	}

	// Get map entity counts
	totalEntities := 0
	entitiesWithMapContext := 0
	entitiesOnActiveMap := 0
	entitiesOnWrongMap := 0
	enemiesOnWrongMap := 0
	enemyMapContextIDs := make(map[ecs.EntityID]int) // Count of enemies per map context

	for _, entity := range g.world.GetAllEntities() {
		if entity.HasTag("map") || entity.HasTag("tilemap") {
			continue // Skip map entities
		}

		totalEntities++

		if g.world.HasComponent(entity.ID, components.MapContextID) {
			entitiesWithMapContext++

			contextComp, _ := g.world.GetComponent(entity.ID, components.MapContextID)
			mapContext := contextComp.(*components.MapContextComponent)

			if mapContext.MapID == activeMap.ID {
				entitiesOnActiveMap++
				if entity.HasTag("enemy") {
					enemyMapContextIDs[mapContext.MapID]++
				}
			} else {
				entitiesOnWrongMap++
				if entity.HasTag("enemy") {
					enemiesOnWrongMap++
					enemyMapContextIDs[mapContext.MapID]++

					// Log detailed info about misplaced enemies
					if g.world.HasComponent(entity.ID, components.Name) {
						nameComp, _ := g.world.GetComponent(entity.ID, components.Name)
						name := nameComp.(*components.NameComponent)
						systems.GetDebugLog().Add(fmt.Sprintf("MISPLACED: Enemy '%s' (ID: %d) on map %d instead of %d",
							name.Name, entity.ID, mapContext.MapID, activeMap.ID))
					}
				}
			}
		}
	}

	// Convert enemy map context IDs map to a string for logging
	var enemyContextsStr string
	for mapID, count := range enemyMapContextIDs {
		// Get map type for this ID if possible
		mapType := "unknown"
		mapEntity := g.world.GetEntity(mapID)
		if mapEntity != nil && g.world.HasComponent(mapID, components.MapType) {
			typeComp, _ := g.world.GetComponent(mapID, components.MapType)
			mapTypeComp := typeComp.(*components.MapTypeComponent)
			mapType = mapTypeComp.MapType
		}

		enemyContextsStr += fmt.Sprintf(" %s(%d):%d", mapType, mapID, count)
	}

	// Add all the debug info
	systems.GetDebugLog().Add(fmt.Sprintf("--- MAP DEBUG INFO ---"))
	systems.GetDebugLog().Add(fmt.Sprintf("Active Map: %s (Level: %d, ID: %d)", mapType, mapLevel, activeMap.ID))
	systems.GetDebugLog().Add(fmt.Sprintf("Player Position: %d,%d (Tile: %d)", playerX, playerY, tileType))
	systems.GetDebugLog().Add(fmt.Sprintf("Player MapContext ID: %d (Matches active: %v)", playerMapID, playerMapID == activeMap.ID))
	systems.GetDebugLog().Add(fmt.Sprintf("Entity Counts - Total: %d, With MapContext: %d, On Active Map: %d",
		totalEntities, entitiesWithMapContext, entitiesOnActiveMap))
	systems.GetDebugLog().Add(fmt.Sprintf("Entities on wrong map: %d, Enemies on wrong map: %d",
		entitiesOnWrongMap, enemiesOnWrongMap))
	if len(enemyContextsStr) > 0 {
		systems.GetDebugLog().Add(fmt.Sprintf("Enemy distribution by map context:%s", enemyContextsStr))
	}
}

// printMapSummary outputs information about all maps in the registry
func (g *Game) printMapSummary() {
	systems.GetDebugLog().Add("--- MAP SUMMARY ---")

	// Get all map entities
	mapEntities := g.world.GetEntitiesWithTag("map")
	worldMapEntities := g.world.GetEntitiesWithTag("worldmap")

	// Combine both lists
	allMapEntities := append(mapEntities, worldMapEntities...)

	// Log each map's details
	for _, mapEntity := range allMapEntities {
		var mapType string = "unknown"
		var mapLevel int = -1

		// Get map type info
		if typeComp, exists := g.world.GetComponent(mapEntity.ID, components.MapType); exists {
			mapTypeComp := typeComp.(*components.MapTypeComponent)
			mapType = mapTypeComp.MapType
			mapLevel = mapTypeComp.Level
		}

		// Get map dimensions
		var mapWidth, mapHeight int
		if mapComp, exists := g.world.GetComponent(mapEntity.ID, components.MapComponentID); exists {
			mc := mapComp.(*components.MapComponent)
			mapWidth = mc.Width
			mapHeight = mc.Height
		}

		systems.GetDebugLog().Add(fmt.Sprintf("Map: %s (Level %d, ID: %d, Size: %dx%d)",
			mapType, mapLevel, mapEntity.ID, mapWidth, mapHeight))
	}

	// Log active map
	activeMap := g.mapRegistrySystem.GetActiveMap()
	if activeMap != nil {
		systems.GetDebugLog().Add(fmt.Sprintf("Active map ID: %d", activeMap.ID))
	}
}
