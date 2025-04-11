package spawners

import (
	"fmt"
	"image/color"
	"strconv"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
)

// EntitySpawner manages the creation of game entities
type EntitySpawner struct {
	world           *ecs.World
	templateManager *data.EntityTemplateManager
	logMessage      func(string) // Function for logging messages
	spawnMapID      ecs.EntityID // ID of the map to spawn entities on
}

// NewEntitySpawner creates a new entity spawner
func NewEntitySpawner(world *ecs.World, templateManager *data.EntityTemplateManager, logFunc func(string)) *EntitySpawner {
	return &EntitySpawner{
		world:           world,
		templateManager: templateManager,
		logMessage:      logFunc,
		spawnMapID:      0, // Initialize to 0 (no active map)
	}
}

// SetSpawnMapID explicitly sets the map ID to use for spawning entities
func (s *EntitySpawner) SetSpawnMapID(mapID ecs.EntityID) {
	s.spawnMapID = mapID
}

// CreatePlayer creates a player entity at the given position
func (s *EntitySpawner) CreatePlayer(x, y int) *ecs.Entity {
	// Create the player entity
	playerEntity := s.world.CreateEntity()
	playerEntity.AddTag("player")
	s.world.TagEntity(playerEntity.ID, "player")

	// Add position component
	s.world.AddComponent(playerEntity.ID, components.Position, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Use position-based tile reference for the player character
	s.world.AddComponent(playerEntity.ID, components.Renderable, components.NewRenderableComponentByPos(
		10, 14, // X,Y position in the tileset
		color.RGBA{255, 255, 255, 255}, // White color
	))

	s.world.AddComponent(playerEntity.ID, components.Player, &components.PlayerComponent{})

	s.world.AddComponent(playerEntity.ID, components.Stats, &components.StatsComponent{
		Health:        100,
		MaxHealth:     100,
		Attack:        5,
		Defense:       2,
		Level:         1,
		Exp:           0,
		HealingFactor: 5,
	})

	s.world.AddComponent(playerEntity.ID, components.Collision, &components.CollisionComponent{
		Blocks: true,
	})

	if s.logMessage != nil {
		s.logMessage("Player created at " + strconv.Itoa(x) + "," + strconv.Itoa(y))
	}

	return playerEntity
}

// CreateCamera creates a camera entity that follows the given target entity
func (s *EntitySpawner) CreateCamera(targetEntityID uint64, x, y int) *ecs.Entity {
	// Create camera entity
	cameraEntity := s.world.CreateEntity()
	cameraEntity.AddTag("camera")
	s.world.TagEntity(cameraEntity.ID, "camera")

	// Create camera component
	cameraComp := components.NewCameraComponent(targetEntityID)

	// Set initial camera position
	cameraComp.X = x - config.GameScreenWidth/2
	cameraComp.Y = y - config.GameScreenHeight/2

	// Add the camera component
	s.world.AddComponent(cameraEntity.ID, components.Camera, cameraComp)

	return cameraEntity
}

// CreateEnemy creates an enemy entity at the given position
func (s *EntitySpawner) CreateEnemy(x, y int, enemyType string) (*ecs.Entity, error) {
	// Try to load the enemy template from our data
	template, exists := s.templateManager.GetTemplate(enemyType)

	if !exists {
		// Return error if template not found
		return nil, fmt.Errorf("no template found for enemy type '%s'", enemyType)
	}

	// Create the enemy entity
	enemyEntity := s.world.CreateEntity()
	enemyEntity.AddTag("enemy")
	s.world.TagEntity(enemyEntity.ID, "enemy")
	enemyEntity.AddTag("ai")
	s.world.TagEntity(enemyEntity.ID, "ai")

	// Add position component
	s.world.AddComponent(enemyEntity.ID, components.Position, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Use the template data for renderable component
	renderable := components.NewRenderableComponentByPos(
		template.TileX,
		template.TileY,
		data.ParseHexColor(template.Color),
	)

	// Use the template data for stats component
	stats := &components.StatsComponent{
		Health:          template.Health,
		MaxHealth:       template.Health,
		Attack:          template.Attack,
		Defense:         template.Defense,
		Level:           template.Level,
		Exp:             template.XP,
		ActionPoints:    template.ActionPoints,
		MaxActionPoints: template.MaxActionPoints,
		Recovery:        template.Recovery,
	}

	// Add any entity-specific tags from the template
	for _, tag := range template.Tags {
		s.world.TagEntity(enemyEntity.ID, tag)
	}
	// Add components
	s.world.AddComponent(enemyEntity.ID, components.Renderable, renderable)
	s.world.AddComponent(enemyEntity.ID, components.Stats, stats)
	s.world.AddComponent(enemyEntity.ID, components.AI, &components.AIComponent{
		Type:       template.AIType,
		SightRange: 8,                       // How far the zombie can see
		Path:       []components.PathNode{}, // Initialize empty path
	})
	// Add name component for display in messages
	s.world.AddComponent(enemyEntity.ID, components.Name, components.NewNameComponent(template.Name))

	// Set collision based on template
	s.world.AddComponent(enemyEntity.ID, components.Collision, &components.CollisionComponent{
		Blocks: template.BlocksPath,
	})

	// Add map context component to associate the enemy with the map
	var mapID ecs.EntityID
	if s.spawnMapID != 0 {
		mapID = s.spawnMapID
		if s.logMessage != nil {
			s.logMessage(fmt.Sprintf("DEBUG: Creating enemy with MapContext ID: %d", mapID))
		}
	} else {
		// Fallback to getting the active map if spawnMapID not set
		mapID = s.getActiveMap()
		if s.logMessage != nil && mapID != 0 {
			s.logMessage(fmt.Sprintf("DEBUG: Creating enemy with fallback MapContext ID: %d", mapID))
		}
	}

	if mapID != 0 {
		s.world.AddComponent(enemyEntity.ID, components.MapContextID, components.NewMapContextComponent(mapID))
	} else if s.logMessage != nil {
		s.logMessage("WARNING: Created enemy with no map context")
	}

	return enemyEntity, nil
}

// CreateTileMapping creates a tile mapping entity with default definitions
func (s *EntitySpawner) CreateTileMapping() *ecs.Entity {
	tileMapEntity := s.world.CreateEntity()
	tileMapEntity.AddTag("tilemap")
	s.world.TagEntity(tileMapEntity.ID, "tilemap")

	// Add the tile mapping component with default definitions
	s.world.AddComponent(tileMapEntity.ID, components.Appearance, components.NewTileMappingComponent())

	return tileMapEntity
}

// getActiveMap returns the currently active map entity (if any)
func (s *EntitySpawner) getActiveMap() ecs.EntityID {
	// Try to find the map registry system first
	for _, system := range s.world.GetSystems() {
		// Check if it's a MapRegistrySystem (string comparison is a simple way to check the type)
		if fmt.Sprintf("%T", system) == "*systems.MapRegistrySystem" {
			// Use reflection to safely call the GetActiveMap method
			if mapRegistry, ok := system.(interface {
				GetActiveMap() *ecs.Entity
			}); ok {
				if activeMap := mapRegistry.GetActiveMap(); activeMap != nil {
					return activeMap.ID
				}
			}
		}
	}

	// Fallback: try to find a map system
	for _, system := range s.world.GetSystems() {
		if fmt.Sprintf("%T", system) == "*systems.MapSystem" {
			if mapSys, ok := system.(interface {
				GetActiveMap() *ecs.Entity
			}); ok {
				if activeMap := mapSys.GetActiveMap(); activeMap != nil {
					return activeMap.ID
				}
			}
		}
	}

	// As a last resort, look for any entity with the "map" tag
	mapEntities := s.world.GetEntitiesWithTag("map")
	if len(mapEntities) > 0 {
		return mapEntities[0].ID
	}

	// No active map found
	return 0
}
