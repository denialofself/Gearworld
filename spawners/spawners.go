package spawners

import (
	"fmt"
	"image/color"
	"strconv"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/systems"
)

// EntitySpawner manages the creation of game entities
type EntitySpawner struct {
	world           *ecs.World
	templateManager *data.EntityTemplateManager
}

// NewEntitySpawner creates a new entity spawner
func NewEntitySpawner(world *ecs.World, templateManager *data.EntityTemplateManager) *EntitySpawner {
	return &EntitySpawner{
		world:           world,
		templateManager: templateManager,
	}
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
		Health:    100,
		MaxHealth: 100,
		Attack:    5,
		Defense:   2,
		Level:     1,
		Exp:       0,
	})

	s.world.AddComponent(playerEntity.ID, components.Collision, &components.CollisionComponent{
		Blocks: true,
	})

	systems.GetMessageLog().Add("Player created at " + strconv.Itoa(x) + "," + strconv.Itoa(y))

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
		Health:    template.Health,
		MaxHealth: template.Health,
		Attack:    template.Attack,
		Defense:   template.Defense,
		Level:     template.Level,
		Exp:       template.XP,
	}

	// Add any entity-specific tags from the template
	for _, tag := range template.Tags {
		s.world.TagEntity(enemyEntity.ID, tag)
	}

	// Add components
	s.world.AddComponent(enemyEntity.ID, components.Renderable, renderable)
	s.world.AddComponent(enemyEntity.ID, components.Stats, stats)

	// Set collision based on template
	s.world.AddComponent(enemyEntity.ID, components.Collision, &components.CollisionComponent{
		Blocks: template.BlocksPath,
	})

	// Set AI type based on template or default to enemy type
	aiType := enemyType
	if template.AIType != "" {
		aiType = template.AIType
	}

	s.world.AddComponent(enemyEntity.ID, components.AI, &components.AIComponent{
		Type: aiType,
	})

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
