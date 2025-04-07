package systems

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"ebiten-rogue/generation"
)

// MapSystem handles map-related operations and rendering
type MapSystem struct {
	dungeonGenerator   *generation.DungeonGenerator
	currentDungeonType generation.DungeonType
	world              *ecs.World
}

// NewMapSystem creates a new map system
func NewMapSystem() *MapSystem {
	return &MapSystem{
		dungeonGenerator:   generation.NewDungeonGenerator(),
		currentDungeonType: generation.DungeonTypeSmallBSP, // Default to small BSP dungeon
	}
}

// Update checks for map-related events and updates
func (s *MapSystem) Update(world *ecs.World, dt float64) {
	// Store world reference for regeneration
	s.world = world

	// Check for F2 key press to toggle dungeon type
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		s.ToggleDungeonType(world)
	}
}

// ToggleDungeonType switches between different dungeon generation types
func (s *MapSystem) ToggleDungeonType(world *ecs.World) {
	// Cycle to the next dungeon type
	switch s.currentDungeonType {
	case generation.DungeonTypeRandom:
		s.currentDungeonType = generation.DungeonTypeSmallBSP
		GetMessageLog().Add("Switched to Small BSP dungeon")
		s.RegenerateMap(world)
	case generation.DungeonTypeSmallBSP:
		s.currentDungeonType = generation.DungeonTypeLargeBSP
		GetMessageLog().Add("Switched to Large BSP dungeon")
		s.RegenerateMap(world)
	case generation.DungeonTypeLargeBSP:
		s.currentDungeonType = generation.DungeonTypeRandom
		GetMessageLog().Add("Switched to Random dungeon")
		s.RegenerateMap(world)
	}
}

// RegenerateMap recreates the map using the current dungeon type
func (s *MapSystem) RegenerateMap(world *ecs.World) {
	// Find and remove the old map entity if it exists
	entities := world.GetEntitiesWithTag("map")
	for _, e := range entities {
		world.RemoveEntity(e.ID)
	}

	// Generate new map based on current type
	var mapEntity *ecs.Entity
	switch s.currentDungeonType {
	case generation.DungeonTypeRandom:
		mapEntity = s.GenerateDungeon(world, 80, 50)
	case generation.DungeonTypeSmallBSP:
		mapEntity = s.GenerateSmallBSPDungeon(world, 80, 50)
	case generation.DungeonTypeLargeBSP:
		mapEntity = s.GenerateLargeBSPDungeon(world, 80*10, 50*10) // Large BSP dungeon (10x10 screens)
	}

	// Also regenerate player at a valid position
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) > 0 {
		player := playerEntities[0]
		mapCompInterface, exists := world.GetComponent(mapEntity.ID, components.MapComponentID)
		if !exists {
			GetMessageLog().Add("Error: Map component not found")
			return
		}
		mapComp := mapCompInterface.(*components.MapComponent)

		// Find empty position for player
		x, y := s.FindEmptyPosition(mapComp)

		// Update player position
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

// GenerateDungeon creates a new procedural dungeon map (for standard non-chunked maps)
func (s *MapSystem) GenerateDungeon(world *ecs.World, width, height int) *ecs.Entity {
	// Create map entity
	mapEntity := world.CreateEntity()
	mapEntity.AddTag("map")
	world.TagEntity(mapEntity.ID, "map")

	// Create map component
	mapComp := components.NewMapComponent(width, height)
	world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Use the DungeonGenerator to build the map
	s.dungeonGenerator.GenerateRoomsAndCorridors(mapComp)

	// Add debug message
	GetMessageLog().Add("Map generated with dimensions " + strconv.Itoa(width) + "x" + strconv.Itoa(height))

	return mapEntity
}

// GenerateSmallBSPDungeon creates a small dungeon using BSP partitioning
func (s *MapSystem) GenerateSmallBSPDungeon(world *ecs.World, width, height int) *ecs.Entity {
	// Create map entity
	mapEntity := world.CreateEntity()
	mapEntity.AddTag("map")
	world.TagEntity(mapEntity.ID, "map")

	// Create map component
	mapComp := components.NewMapComponent(width, height)
	world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Use the DungeonGenerator to build the small BSP dungeon
	s.dungeonGenerator.GenerateSmallBSPDungeon(mapComp)

	// Add debug message
	GetMessageLog().Add("Small BSP dungeon generated with dimensions " + strconv.Itoa(width) + "x" + strconv.Itoa(height))

	return mapEntity
}

// GenerateLargeBSPDungeon creates a large dungeon using BSP partitioning
func (s *MapSystem) GenerateLargeBSPDungeon(world *ecs.World, width, height int) *ecs.Entity {
	// Create map entity
	mapEntity := world.CreateEntity()
	mapEntity.AddTag("map")
	world.TagEntity(mapEntity.ID, "map")

	// Create map component
	mapComp := components.NewMapComponent(width, height)
	world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Use the DungeonGenerator to build the large BSP dungeon
	s.dungeonGenerator.GenerateLargeBSPDungeon(mapComp)

	// Add debug message
	GetMessageLog().Add("Large BSP dungeon generated with dimensions " + strconv.Itoa(width) + "x" + strconv.Itoa(height))

	return mapEntity
}

// FindEmptyPosition locates an unoccupied floor tile in the map
func (s *MapSystem) FindEmptyPosition(mapComp *components.MapComponent) (int, int) {
	return s.dungeonGenerator.FindEmptyPosition(mapComp)
}
