package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"math/rand"
	"time"
)

// MapSystem handles map-related operations and rendering
type MapSystem struct {
	world *ecs.World
}

// NewMapSystem creates a new map system
func NewMapSystem() *MapSystem {
	return &MapSystem{}
}

// Update checks for map-related events and updates
func (s *MapSystem) Update(world *ecs.World, dt float64) {
	// Store world reference for operations that might need it
	s.world = world

	// Map system now focuses only on map management, not generation
	// Key handling for map type switching is moved to Game
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
