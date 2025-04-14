package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
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

// findTransitionDestination finds the appropriate landing spot on the target map
func (s *MapSystem) findTransitionDestination(mapComp *components.MapComponent, tileType int) (int, int) {
	// Determine which type of stairs to look for on the target map
	targetStairType := components.TileStairsUp
	if tileType == components.TileStairsUp {
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
