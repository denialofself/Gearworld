package main

import (
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
	"ebiten-rogue/screens"
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
	itemSpawner               *spawners.ItemSpawner
	aiPathfindingSystem       *systems.AIPathfindingSystem
	aiTurnProcessorSystem     *systems.AITurnProcessorSystem
	effectsSystem             *systems.EffectsSystem
	inventorySystem           *systems.InventorySystem
	equipmentSystem           *systems.EquipmentSystem
	fovSystem                 *systems.FOVSystem
	screenStack               *screens.ScreenStack
	audioSystem               *systems.AudioSystem
	containerSystem           *systems.ContainerSystem
	deathSystem               *systems.DeathSystem
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
	equipmentSystem := systems.NewEquipmentSystem()
	fovSystem := systems.NewFOVSystem()
	containerSystem := systems.NewContainerSystem(world)
	deathSystem := systems.NewDeathSystem()

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

	// Load container templates
	err = templateManager.LoadContainerTemplatesFromDirectory("data/containers")
	if err != nil {
		fmt.Printf("Warning: Failed to load container templates: %v\n", err)
	}

	// Create entity spawner
	entitySpawner := spawners.NewEntitySpawner(world, templateManager, systems.GetMessageLog().Add)

	// Create item spawner
	itemSpawner := spawners.NewItemSpawner(world, templateManager)

	// Create audio system first since it needs to be shared
	audioSystem := systems.NewAudioSystem()

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
	world.AddSystem(equipmentSystem)
	world.AddSystem(fovSystem)
	world.AddSystem(containerSystem)
	world.AddSystem(deathSystem)
	world.AddSystem(renderSystem) // Render system should be last to see all changes

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
		itemSpawner:               itemSpawner,
		aiPathfindingSystem:       aiPathfindingSystem,
		aiTurnProcessorSystem:     aiTurnProcessorSystem,
		effectsSystem:             effectsSystem,
		inventorySystem:           inventorySystem,
		equipmentSystem:           equipmentSystem,
		fovSystem:                 fovSystem,
		screenStack:               screens.NewScreenStack(),
		audioSystem:               audioSystem,
		containerSystem:           containerSystem,
		deathSystem:               deathSystem,
	}

	// Initialize event listeners
	movementSystem.Initialize(world)
	combatSystem.Initialize(world)
	aiPathfindingSystem.Initialize(world)
	aiTurnProcessorSystem.Initialize(world)
	effectsSystem.Initialize(world)
	inventorySystem.Initialize(world)
	equipmentSystem.Initialize(world)
	fovSystem.Initialize(world)
	renderSystem.Initialize(world)
	containerSystem.Initialize(world)
	deathSystem.Initialize(world)

	// Push the start screen onto the stack
	game.screenStack.Push(screens.NewStartScreen(audioSystem))

	return game
}

// Update updates the game state.
func (g *Game) Update() error {
	// Get the current screen
	currentScreen := g.screenStack.Peek()

	// Handle screen transitions
	switch screen := currentScreen.(type) {
	case *screens.StartScreen:
		// Update the start screen
		if err := screen.Update(); err != nil {
			switch err {
			case screens.ErrNewGame:
				// Stop the background music
				g.audioSystem.StopBGM()

				// Initialize the game world
				g.initialize()

				// Create and push the game screen
				gameScreen := screens.NewGameScreen(
					g.world,
					g.renderSystem,
					g.mapSystem,
					g.mapRegistrySystem,
					g.movementSystem,
					g.playerTurnProcessorSystem,
					g.combatSystem,
					g.cameraSystem,
					g.aiPathfindingSystem,
					g.aiTurnProcessorSystem,
					g.effectsSystem,
					g.inventorySystem,
					g.equipmentSystem,
					g.fovSystem,
					g.containerSystem,
					g.audioSystem,
					g.deathSystem,
				)

				// Pop the start screen and push the game screen
				g.screenStack.Pop()
				g.screenStack.Push(gameScreen)
			case screens.ErrLoadGame:
				// TODO: Implement load game functionality
				systems.GetMessageLog().Add("Load game not implemented yet")
			case screens.ErrOptions:
				// TODO: Implement options screen
				systems.GetMessageLog().Add("Options not implemented yet")
			case screens.ErrQuit:
				return ebiten.Termination
			}
		}
	case *screens.GameScreen:
		// Check for game over event
		g.world.GetEventManager().Subscribe(systems.EventGameOver, func(event ecs.Event) {
			// Pop the game screen and push the game over screen
			g.screenStack.Pop()
			g.screenStack.Push(screens.NewGameOverScreen())
		})
	case *screens.GameOverScreen:
		// Return to start screen on Escape key
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			// Stop any background music
			g.audioSystem.StopBGM()

			// Log current state before cleanup
			systems.GetDebugLog().Add("=== GAME OVER CLEANUP START ===")
			activeMap := g.mapRegistrySystem.GetActiveMap()
			if activeMap != nil {
				systems.GetDebugLog().Add(fmt.Sprintf("Current active map ID: %d", activeMap.ID))
			} else {
				systems.GetDebugLog().Add("No active map")
			}

			// Clear the map registry
			g.mapRegistrySystem.Clear()
			systems.GetDebugLog().Add("Map registry cleared")

			// Reinitialize the game
			systems.GetDebugLog().Add("Reinitializing game...")
			g.initialize()
			systems.GetDebugLog().Add("Game reinitialized")

			// Pop the game over screen and push the start screen
			systems.GetDebugLog().Add("Popping game over screen and pushing start screen")
			g.screenStack.Pop()
			g.screenStack.Push(screens.NewStartScreen(g.audioSystem))
			systems.GetDebugLog().Add("=== GAME OVER CLEANUP COMPLETE ===")
		}
	}

	// Update the current screen
	return g.screenStack.Update()
}

// Draw draws the game screen.
func (g *Game) Draw(screen *ebiten.Image) {
	g.screenStack.Draw(screen)
}

// Layout implements ebiten.Game's Layout.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenStack.Layout(outsideWidth, outsideHeight)
}

// initialize sets up the initial game state
func (g *Game) initialize() {
	// Clear the world and map registry
	systems.GetDebugLog().Add("Clearing world and map registry...")

	// Reset the entity ID counter
	ecs.ResetEntityID()

	// Remove all entities from the world
	entities := g.world.GetAllEntities()
	for _, entity := range entities {
		g.world.RemoveEntity(entity.ID)
	}

	g.mapRegistrySystem.Clear()
	systems.GetDebugLog().Add("World and map registry cleared")

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
	systems.GetDebugLog().Add(fmt.Sprintf("Created world map with ID: %d", worldMapEntity.ID))

	// Register the world map with the map registry
	g.mapRegistrySystem.RegisterMap(worldMapEntity)

	// Create a dungeon themer
	dungeonThemer := generation.NewDungeonThemer(
		g.world,
		g.templateManager,
		g.entitySpawner,
		systems.GetMessageLog().Add,
	)

	// Load themes from the data/themes directory
	err := dungeonThemer.LoadThemesFromDirectory("data/themes")
	if err != nil {
		systems.GetMessageLog().Add(fmt.Sprintf("Error loading dungeon themes: %v", err))
	}

	// Configure the dungeon (level 1, abandoned theme, large size)
	config := generation.DungeonConfiguration{
		Level:         1,
		Size:          generation.SizeSmall,
		Generator:     generation.GeneratorBSP,
		AddStairsUp:   true,               // Add stairs up to return to the world map
		ThemeID:       "starting_station", // Use the JSON theme if available
		DensityFactor: 1.0,                // Standard monster density
	}

	// Generate the themed dungeon with appropriate monsters
	dungeonFloors := dungeonThemer.GenerateThemedDungeon(config)
	if len(dungeonFloors) == 0 {
		systems.GetDebugLog().Add("Error: No dungeon floors were generated")
		return
	}

	// Register all dungeon floors with the map registry
	for i, floorEntity := range dungeonFloors {
		floorLevel := i + 1 // Floor levels are 1-based
		// Add map type component if it doesn't exist
		if !g.world.HasComponent(floorEntity.ID, components.MapType) {
			g.world.AddComponent(floorEntity.ID, components.MapType,
				components.NewMapTypeComponent("dungeon", floorLevel))
		}

		// Log the dungeon entity ID for debugging
		systems.GetDebugLog().Add(fmt.Sprintf("Created dungeon floor %d with ID: %d", floorLevel, floorEntity.ID))

		// Register the dungeon floor with the map registry
		g.mapRegistrySystem.RegisterMap(floorEntity)
	}

	// Get the first floor entity (where the player starts)
	startingFloorEntity := dungeonFloors[0]

	// Get the map component from the dungeon entity
	var mapComp *components.MapComponent
	if comp, exists := g.world.GetComponent(startingFloorEntity.ID, components.MapComponentID); exists {
		mapComp = comp.(*components.MapComponent)
	}

	if mapComp == nil {
		systems.GetDebugLog().Add("Error: Failed to get map component")
		return
	}

	// We'll start in the dungeon
	// Set the active map in the map registry system
	g.mapRegistrySystem.SetActiveMap(startingFloorEntity)
	systems.GetDebugLog().Add(fmt.Sprintf("Set active map to dungeon floor 1 with ID: %d", startingFloorEntity.ID))

	// Find empty position for player
	playerX, playerY := g.mapSystem.FindEmptyPosition(mapComp)

	// Create the player entity
	playerEntity := g.entitySpawner.CreatePlayer(playerX, playerY)

	// Add map context component to the player
	g.world.AddComponent(playerEntity.ID, components.MapContextID,
		components.NewMapContextComponent(startingFloorEntity.ID))

	// Create starter chest next to player
	chestX, chestY := playerX+1, playerY
	g.itemSpawner.SetSpawnMapID(startingFloorEntity.ID)
	g.itemSpawner.CreateContainer(chestX, chestY, "starter_chest")

	// Create a camera entity for the player
	g.entitySpawner.CreateCamera(uint64(playerEntity.ID), playerX, playerY)

	// Print a summary of all maps and their IDs
	g.printMapSummary()

	// Add welcome message
	systems.GetMessageLog().Add("Welcome to the dungeon! Use arrow keys to move.")
	systems.GetMessageLog().AddEnvironment("You awaken in a cracked cryogenic pod, the walls of the pod are covered in frost.")
	systems.GetMessageLog().AddEnvironment("The chamber is dimly lit, and something scuttles in the dark")
}

// Flag to track if we need to redraw the screen
var needsRedraw = true

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
