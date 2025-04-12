package systems

import (
	"math"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// FOVSystem handles field of vision calculations
type FOVSystem struct{}

// NewFOVSystem creates a new FOV system
func NewFOVSystem() *FOVSystem {
	return &FOVSystem{}
}

// Update calculates FOV for entities with FOV components
func (s *FOVSystem) Update(world *ecs.World, dt float64) {
	// Find the active map
	var activeMap *ecs.Entity
	var activeMapRegistrySystem *MapRegistrySystem

	// Find the MapRegistrySystem
	for _, system := range world.GetSystems() {
		if mapRegistry, ok := system.(*MapRegistrySystem); ok {
			activeMapRegistrySystem = mapRegistry
			break
		}
	}

	// Get the active map from the registry system
	if activeMapRegistrySystem != nil {
		activeMap = activeMapRegistrySystem.GetActiveMap()
	}

	// If no active map found, return
	if activeMap == nil {
		return
	}

	// Get the map component
	var mapComp *components.MapComponent
	if comp, exists := world.GetComponent(activeMap.ID, components.MapComponentID); exists {
		mapComp = comp.(*components.MapComponent)
	} else {
		// No map component found
		return
	}

	// Reset visibility for all map tiles
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.Visible[y][x] = false
		}
	}

	// Process entities with FOV components
	for _, entity := range world.GetEntitiesWithComponent(components.FOV) {
		// Only process entities on the active map
		if !s.entityIsOnActiveMap(world, entity.ID, activeMap.ID) {
			continue
		}

		// Get position component
		var pos *components.PositionComponent
		if posComp, exists := world.GetComponent(entity.ID, components.Position); exists {
			pos = posComp.(*components.PositionComponent)
		} else {
			continue // No position, can't calculate FOV
		}

		// Get FOV component
		var fov *components.FOVComponent
		if fovComp, exists := world.GetComponent(entity.ID, components.FOV); exists {
			fov = fovComp.(*components.FOVComponent)
		} else {
			continue // No FOV component
		}

		// Calculate visibility
		s.calculateFOV(world, mapComp, pos.X, pos.Y, fov.Range)

		// If this entity is a player, mark visible tiles as explored
		if entity.HasTag("player") {
			for y := 0; y < mapComp.Height; y++ {
				for x := 0; x < mapComp.Width; x++ {
					if mapComp.Visible[y][x] {
						mapComp.Explored[y][x] = true
					}
				}
			}
		}
	}
}

// entityIsOnActiveMap checks if an entity is on the active map
func (s *FOVSystem) entityIsOnActiveMap(world *ecs.World, entityID, activeMapID ecs.EntityID) bool {
	if comp, exists := world.GetComponent(entityID, components.MapContext); exists {
		mapContext := comp.(*components.MapContextComponent)
		return mapContext.MapID == activeMapID
	}
	return false // No map context, assume not on active map
}

// calculateFOV calculates what tiles are visible from a given position
// This implements a basic raycasting FOV algorithm
func (s *FOVSystem) calculateFOV(world *ecs.World, mapComp *components.MapComponent, x, y, radius int) {
	// The origin is always visible
	mapComp.Visible[y][x] = true
	mapComp.Explored[y][x] = true

	// Cast rays in a full circle
	for angle := 0; angle < 360; angle++ {
		s.castRay(mapComp, x, y, radius, float64(angle)*(math.Pi/180.0))
	}
}

// castRay casts a single ray from origin and marks tiles it passes through as visible
func (s *FOVSystem) castRay(mapComp *components.MapComponent, x, y, radius int, angle float64) {
	// Calculate the direction vector
	dx := math.Cos(angle)
	dy := math.Sin(angle)

	// Start at the origin
	currentX := float64(x)
	currentY := float64(y)

	// Cast the ray outward to the radius
	for i := 0; i < radius; i++ {
		// Move along the ray
		currentX += dx
		currentY += dy

		// Convert to integer coordinates
		tileX := int(math.Floor(currentX))
		tileY := int(math.Floor(currentY))

		// Skip if out of bounds
		if tileX < 0 || tileX >= mapComp.Width || tileY < 0 || tileY >= mapComp.Height {
			continue
		}

		// Mark this tile as visible
		mapComp.Visible[tileY][tileX] = true

		// Stop if we hit a wall
		if mapComp.IsWall(tileX, tileY) {
			break
		}
	}
}

// Initialize sets up event listeners
func (s *FOVSystem) Initialize(world *ecs.World) {
	// Register to listen for events that should trigger FOV updates
	world.RegisterEventListener(func(w *ecs.World, event interface{}) {
		// Check for turn completed events
		if _, ok := event.(TurnCompletedEvent); ok {
			s.Update(w, 0)
			return
		}

		// Check for player movement events
		if _, ok := event.(PlayerMoveEvent); ok {
			s.Update(w, 0)
			return
		}
	})
}
