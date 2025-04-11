package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
	"math/rand"
	"time"
)

// MapSystem handles map-related operations and rendering
type MapSystem struct {
	world     *ecs.World
	activeMap *ecs.Entity
}

// NewMapSystem creates a new map system
func NewMapSystem() *MapSystem {
	return &MapSystem{}
}

// Update checks for map-related events and updates
func (s *MapSystem) Update(world *ecs.World, dt float64) {
	// Store world reference for operations that might need it
	s.world = world

	// Check for map transition events
	s.handleMapTransitions(world)
}

// SetActiveMap sets the currently active map
func (s *MapSystem) SetActiveMap(mapEntity *ecs.Entity) {
	s.activeMap = mapEntity
}

// GetActiveMap returns the currently active map entity
func (s *MapSystem) GetActiveMap() *ecs.Entity {
	return s.activeMap
}

// handleMapTransitions processes transitions between maps when player interacts with stairs
func (s *MapSystem) handleMapTransitions(world *ecs.World) {
	// DISABLED - Transitions are now handled by MapRegistrySystem
	// This stub is kept for compatibility purposes
	return
}

// RepositionPlayer places the player at a new empty position on the map
func (s *MapSystem) RepositionPlayer(world *ecs.World, mapEntity *ecs.Entity) {
	// Get the map component
	mapCompInterface, exists := world.GetComponent(mapEntity.ID, components.MapComponentID)
	if !exists {
		GetMessageLog().Add("Error: Map component not found")
		return
	}
	mapComp := mapCompInterface.(*components.MapComponent)

	// Find empty position for player
	x, y := s.FindEmptyPosition(mapComp)

	// Update player position
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) > 0 {
		player := playerEntities[0]
		posCompInterface, exists := world.GetComponent(player.ID, components.Position)
		if !exists {
			GetMessageLog().Add("Error: Player position component not found")
			return
		}
		posComp := posCompInterface.(*components.PositionComponent)
		posComp.X = x
		posComp.Y = y
	}
}

// FindEmptyPosition locates an unoccupied floor tile in the map
func (s *MapSystem) FindEmptyPosition(mapComp *components.MapComponent) (int, int) {
	// Get a random position using a more robust approach
	maxAttempts := 100

	// Create a list of valid floor tiles
	var floorTiles [][2]int

	// First, collect all floor tiles
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				floorTiles = append(floorTiles, [2]int{x, y})
			}
		}
	}

	// If we found floor tiles, return a random one
	if len(floorTiles) > 0 {
		// Use time to seed the random number generator
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomIndex := r.Intn(len(floorTiles))
		return floorTiles[randomIndex][0], floorTiles[randomIndex][1]
	}

	// Try a random approach as fallback
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < maxAttempts; i++ {
		x := r.Intn(mapComp.Width)
		y := r.Intn(mapComp.Height)
		if mapComp.Tiles[y][x] == components.TileFloor {
			return x, y
		}
	}

	// Last resort fallback - return center of map if no floor found
	return mapComp.Width / 2, mapComp.Height / 2
}

// transitionBetweenMaps handles player movement between world map and dungeons
// DEPRECATED: This function is no longer used. Map transitions are now handled by MapRegistrySystem.
func (s *MapSystem) transitionBetweenMaps(world *ecs.World, tileType int, playerPos *components.PositionComponent) {
	// Get the current map type
	mapTypeInterface, exists := world.GetComponent(s.activeMap.ID, components.MapType)
	if !exists {
		GetMessageLog().Add("Error: Cannot determine current map type")
		return
	}
	currentMapType := mapTypeInterface.(*components.MapTypeComponent)

	var targetMapType string
	var targetMapEntity *ecs.Entity
	var targetX, targetY int

	// Determine target map based on current map and tile type
	if currentMapType.MapType == "worldmap" && tileType == components.TileStairsDown {
		// Transition from world map to dungeon
		targetMapType = "dungeon"
		targetMapEntity = s.getDungeonMap(world)
		if targetMapEntity == nil {
			// No dungeon exists yet, create one
			targetMapEntity = s.createNewDungeon(world, true) // true = add stairs up
		}
	} else if currentMapType.MapType == "dungeon" && tileType == components.TileStairsUp {
		// Transition from dungeon to world map
		targetMapType = "worldmap"
		targetMapEntity = s.getWorldMap(world)
		if targetMapEntity == nil {
			// No world map exists yet, create one
			targetMapEntity = s.createWorldMap(world)
		}
	} else {
		// Invalid transition
		GetMessageLog().Add("You can't go that way.")
		return
	}

	// Find destination coordinates on the target map
	targetX, targetY = s.findTransitionDestination(world, targetMapEntity, tileType)
	// Set player position on the new map
	playerPos.X = targetX
	playerPos.Y = targetY

	// Set the active map to the target map
	s.SetActiveMap(targetMapEntity)

	// Update player's map context to match the new map
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) > 0 {
		playerEntity := playerEntities[0]
		if world.HasComponent(playerEntity.ID, components.MapContextID) {
			mapContextComp, _ := world.GetComponent(playerEntity.ID, components.MapContextID)
			mapContext := mapContextComp.(*components.MapContextComponent)
			mapContext.MapID = targetMapEntity.ID
		} else {
			world.AddComponent(playerEntity.ID, components.MapContextID, components.NewMapContextComponent(targetMapEntity.ID))
		}
	}

	// Update camera to center on player
	s.updateCameraPosition(world, targetX, targetY)

	// Log the transition
	if targetMapType == "worldmap" {
		GetMessageLog().Add("You climb the stairs and emerge onto the surface.")
	} else {
		GetMessageLog().Add("You descend into the darkness below.")
	}
}

// getDungeonMap returns the first dungeon map entity, or nil if none exists
func (s *MapSystem) getDungeonMap(world *ecs.World) *ecs.Entity {
	dungeonMaps := world.GetEntitiesWithTag("map") // Regular maps are dungeons
	if len(dungeonMaps) > 0 {
		return dungeonMaps[0]
	}
	return nil
}

// getWorldMap returns the world map entity, or nil if it doesn't exist
func (s *MapSystem) getWorldMap(world *ecs.World) *ecs.Entity {
	worldMaps := world.GetEntitiesWithTag("worldmap")
	if len(worldMaps) > 0 {
		return worldMaps[0]
	}
	return nil
}

// createWorldMap generates a new world map
func (s *MapSystem) createWorldMap(world *ecs.World) *ecs.Entity {
	// Create a world map generator with a random seed
	seed := time.Now().UnixNano()
	worldMapGen := generation.NewWorldMapGenerator(seed)

	// Create a world map with default size
	width, height := 200, 200 // Large world map
	mapEntity := worldMapGen.CreateWorldMapEntity(world, width, height)

	// Add map type component
	world.AddComponent(mapEntity.ID, components.MapType, components.NewMapTypeComponent("worldmap", 0))

	GetMessageLog().Add("A vast wasteland stretches before you...")
	return mapEntity
}

// createNewDungeon generates a new dungeon map with an option to include stairs up
func (s *MapSystem) createNewDungeon(world *ecs.World, addStairsUp bool) *ecs.Entity {
	// Get the dungeon themer from the game
	// This is a placeholder - in a full implementation you would
	// access the dungeon generator from the game instance

	// For now, we'll create a basic dungeon
	GetMessageLog().Add("Generating a new dungeon level...")

	// Create the map entity
	mapEntity := world.CreateEntity()
	mapEntity.AddTag("map")
	world.TagEntity(mapEntity.ID, "map")

	// Create map component with standard dungeon size
	width, height := 80, 45
	mapComp := components.NewMapComponent(width, height)
	world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Use the dungeon generator to create the dungeon
	// This is simplified - we would normally use the DungeonThemer
	dungeonGen := generation.NewDungeonGenerator()
	dungeonGen.SetSeed(time.Now().UnixNano())
	dungeonGen.GenerateBSPDungeon(mapComp)

	// Add map type component
	world.AddComponent(mapEntity.ID, components.MapType, components.NewMapTypeComponent("dungeon", 1))

	// If requested, add stairs up back to the world map
	if addStairsUp {
		s.addStairsUp(mapComp)
	}

	return mapEntity
}

// addStairsUp adds a set of stairs up to a dungeon map, preferably near the player spawn
func (s *MapSystem) addStairsUp(mapComp *components.MapComponent) {
	// Find a suitable position for the stairs up
	// For the test case, place them near where the player will spawn
	x, y := s.FindEmptyPosition(mapComp)

	// Place stairs up at this position
	mapComp.SetTile(x, y, components.TileStairsUp)
}

// findTransitionDestination finds the appropriate landing spot on the target map
func (s *MapSystem) findTransitionDestination(world *ecs.World, targetMap *ecs.Entity, sourceStairType int) (int, int) {
	targetMapComp, exists := world.GetComponent(targetMap.ID, components.MapComponentID)
	if !exists {
		// If we can't get the map, use a default position
		return 1, 1
	}

	mapComp := targetMapComp.(*components.MapComponent)

	// Determine which type of stairs to look for on the target map
	targetStairType := components.TileStairsUp
	if sourceStairType == components.TileStairsUp {
		targetStairType = components.TileStairsDown
	}

	// Look for matching stairs on the target map
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == targetStairType {
				// Found matching stairs
				return x, y
			}
		}
	}

	// If no matching stairs found, find an empty spot
	return s.FindEmptyPosition(mapComp)
}

// updateCameraPosition centers the camera on the given position
func (s *MapSystem) updateCameraPosition(world *ecs.World, x, y int) {
	cameraEntities := world.GetEntitiesWithTag("camera")
	if len(cameraEntities) == 0 {
		return
	}

	cameraEntity := cameraEntities[0]
	cameraComp, exists := world.GetComponent(cameraEntity.ID, components.Camera)
	if !exists {
		return
	}

	camera := cameraComp.(*components.CameraComponent)
	camera.X = x - (config.GameScreenWidth / 2)
	camera.Y = y - (config.GameScreenHeight / 2)

	// Ensure camera doesn't go out of bounds
	mapComp, exists := world.GetComponent(s.activeMap.ID, components.MapComponentID)
	if !exists {
		return
	}
	mapData := mapComp.(*components.MapComponent)

	// Clamp camera X position
	maxX := mapData.Width - config.GameScreenWidth
	if camera.X < 0 {
		camera.X = 0
	} else if camera.X > maxX {
		camera.X = maxX
	}

	// Clamp camera Y position
	maxY := mapData.Height - config.GameScreenHeight
	if camera.Y < 0 {
		camera.Y = 0
	} else if camera.Y > maxY {
		camera.Y = maxY
	}
}
