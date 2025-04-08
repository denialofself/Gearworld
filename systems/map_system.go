package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
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
	// Simple implementation to find an empty position
	// This could be enhanced with more sophisticated algorithms as needed
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				return x, y
			}
		}
	}

	// Fallback - return center of map if no floor found
	return mapComp.Width / 2, mapComp.Height / 2
}
