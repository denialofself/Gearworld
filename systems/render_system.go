package systems

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
)

// RenderSystem handles drawing entities to the screen
type RenderSystem struct {
	tileset      *Tileset
	cameraSystem *CameraSystem // Reference to the camera system
}

// NewRenderSystem creates a new rendering system
func NewRenderSystem(tileset *Tileset) *RenderSystem {
	return &RenderSystem{
		tileset:      tileset,
		cameraSystem: nil, // Will be set via SetCameraSystem
	}
}

// SetCameraSystem sets the camera system to be used for rendering
func (s *RenderSystem) SetCameraSystem(cameraSystem *CameraSystem) {
	s.cameraSystem = cameraSystem
}

// Update renders entities with Position and Renderable components
func (s *RenderSystem) Update(world *ecs.World, dt float64) {
	// If we haven't been given a camera system, find one in the world
	if s.cameraSystem == nil {
		// Find a camera system by iterating through the world's systems
		for _, system := range world.GetSystems() {
			if cameraSystem, ok := system.(*CameraSystem); ok {
				s.cameraSystem = cameraSystem
				break
			}
		}
	}
}

// Draw renders all entities with position and renderable components
func (s *RenderSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Clear the screen
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw the game area (map)
	s.drawGameScreen(world, screen)

	// Draw UI elements
	s.drawStatsPanel(world, screen)
	s.drawMessagesPanel(screen)
}

// drawGameScreen draws the game map and entities
func (s *RenderSystem) drawGameScreen(world *ecs.World, screen *ebiten.Image) {
	// Get the tile mapping entity first
	tileMapEntities := world.GetEntitiesWithTag("tilemap")
	if len(tileMapEntities) == 0 {
		GetMessageLog().Add("Error: No tile mapping entity found")
		return
	}

	// Get the tile mapping component
	var tileMapping *components.TileMappingComponent
	if comp, exists := world.GetComponent(tileMapEntities[0].ID, components.Appearance); exists {
		tileMapping = comp.(*components.TileMappingComponent)
	} else {
		GetMessageLog().Add("Error: No tile mapping component found")
		return
	}

	// Get camera position
	var cameraX, cameraY int
	cameraEntities := world.GetEntitiesWithTag("camera")
	if len(cameraEntities) > 0 {
		if camera, exists := world.GetComponent(cameraEntities[0].ID, components.Camera); exists {
			cameraComp := camera.(*components.CameraComponent)
			cameraX = cameraComp.X
			cameraY = cameraComp.Y
		}
	}

	// Find the standard map entity
	standardMapEntities := world.GetEntitiesWithTag("map")
	if len(standardMapEntities) > 0 {
		s.drawStandardMap(world, screen, standardMapEntities[0].ID, tileMapping, cameraX, cameraY)
	} else {
		GetMessageLog().Add("Error: No map entity found")
		return
	}

	// Draw all entities
	s.drawEntities(world, screen, cameraX, cameraY)
}

// drawStandardMap draws a standard non-chunked map
func (s *RenderSystem) drawStandardMap(world *ecs.World, screen *ebiten.Image, mapID ecs.EntityID,
	tileMapping *components.TileMappingComponent, cameraX, cameraY int) {
	// Get map component
	mapComp, exists := world.GetComponent(mapID, components.MapComponentID)
	if !exists {
		GetMessageLog().Add("Error: Map component not found")
		return
	}
	mapData := mapComp.(*components.MapComponent)

	// Draw map tiles that are visible in the viewport
	for y := 0; y < config.GameScreenHeight; y++ {
		for x := 0; x < config.GameScreenWidth; x++ {
			// Convert screen position to world position
			worldX := x + cameraX
			worldY := y + cameraY

			// Skip if out of map bounds
			if worldX < 0 || worldX >= mapData.Width || worldY < 0 || worldY >= mapData.Height {
				continue
			}

			// Get tile type at this world position
			tileType := mapData.Tiles[worldY][worldX]

			// Get the tile's visual definition from the mapping
			tileDef := tileMapping.GetTileDefinition(tileType)

			// Draw the tile using either position or glyph based on the definition
			if tileDef.UseTilePos {
				// Use direct position reference
				tileID := NewTileID(tileDef.TileX, tileDef.TileY)
				s.tileset.DrawTileByID(screen, tileID, x, y, tileDef.FG)
			} else {
				// Use character-based reference
				s.tileset.DrawTile(screen, tileDef.Glyph, x, y, tileDef.FG)
			}
		}
	}
}

// drawEntities draws all visible entities
func (s *RenderSystem) drawEntities(world *ecs.World, screen *ebiten.Image, cameraX, cameraY int) {
	// Get all entities to render
	for _, entity := range world.GetAllEntities() {
		// Skip map and tilemap entities since we handle those separately
		if entity.HasTag("map") || entity.HasTag("tilemap") {
			continue
		}

		// Only render entities that have both Position and Renderable components
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		rendComp, hasRend := world.GetComponent(entity.ID, components.Renderable)

		if hasPos && hasRend {
			pos := posComp.(*components.PositionComponent)
			rend := rendComp.(*components.RenderableComponent)

			// Convert world position to screen position
			screenX := pos.X - cameraX
			screenY := pos.Y - cameraY

			// Only draw entities within the visible game screen
			if screenX >= 0 && screenX < config.GameScreenWidth &&
				screenY >= 0 && screenY < config.GameScreenHeight {
				// Draw the entity using either position or glyph based approach
				if rend.UseTilePos {
					// Use position-based reference
					tileID := NewTileID(rend.TileX, rend.TileY)
					s.tileset.DrawTileByID(screen, tileID, screenX, screenY, rend.FG)
				} else {
					// Use character-based reference
					s.tileset.DrawTile(screen, rend.Char, screenX, screenY, rend.FG)
				}
			}
		}
	}
}

// drawStatsPanel draws the player stats panel
func (s *RenderSystem) drawStatsPanel(world *ecs.World, screen *ebiten.Image) {
	// Calculate stats panel width
	statsPanelWidth := config.ScreenWidth - config.GameScreenWidth

	// Draw stats panel border and background
	for y := 0; y < config.GameScreenHeight; y++ {
		// Draw vertical border
		s.tileset.DrawTile(screen, '|', config.GameScreenWidth, y, color.RGBA{200, 200, 200, 255})

		// Draw background for better readability (optional dark background)
		for x := config.GameScreenWidth + 1; x < config.ScreenWidth; x++ {
			s.tileset.DrawTile(screen, ' ', x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}

	playerID := playerEntities[0].ID

	// Draw panel title
	s.tileset.DrawString(screen, "CHARACTER INFO", config.GameScreenWidth+2, 1, color.RGBA{255, 255, 255, 255})
	// Draw horizontal separator under title
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 2, color.RGBA{180, 180, 180, 255})
	}

	// Get player stats
	var stats *components.StatsComponent
	if comp, exists := world.GetComponent(playerID, components.Stats); exists {
		stats = comp.(*components.StatsComponent)

		// Draw player stats section
		s.tileset.DrawString(screen, "STATS", config.GameScreenWidth+2, 4, color.RGBA{255, 230, 150, 255})

		// Health with numerical and bar representation
		healthText := "Health: " + strconv.Itoa(stats.Health) + "/" + strconv.Itoa(stats.MaxHealth)
		s.tileset.DrawString(screen, healthText, config.GameScreenWidth+2, 6, color.RGBA{255, 200, 200, 255})

		// Draw health bar
		healthBarWidth := statsPanelWidth - 4 // Leave some margin
		healthPercentage := float64(stats.Health) / float64(stats.MaxHealth)
		filledWidth := int(float64(healthBarWidth) * healthPercentage)

		// Draw the filled portion of the bar
		for x := 0; x < filledWidth; x++ {
			s.tileset.DrawTile(screen, '█', config.GameScreenWidth+2+x, 7, color.RGBA{200, 0, 0, 255})
		}
		// Draw the empty portion of the bar
		for x := filledWidth; x < healthBarWidth; x++ {
			s.tileset.DrawTile(screen, '░', config.GameScreenWidth+2+x, 7, color.RGBA{100, 0, 0, 255})
		}

		// Other stats
		s.tileset.DrawString(screen,
			"Attack:  "+strconv.Itoa(stats.Attack),
			config.GameScreenWidth+2, 9, color.RGBA{200, 200, 255, 255})
		s.tileset.DrawString(screen,
			"Defense: "+strconv.Itoa(stats.Defense),
			config.GameScreenWidth+2, 10, color.RGBA{200, 255, 200, 255})
		s.tileset.DrawString(screen,
			"Level:   "+strconv.Itoa(stats.Level),
			config.GameScreenWidth+2, 11, color.RGBA{255, 255, 200, 255})
		s.tileset.DrawString(screen,
			"EXP:     "+strconv.Itoa(stats.Exp),
			config.GameScreenWidth+2, 12, color.RGBA{200, 200, 255, 255})
	}

	// Get player position
	var position *components.PositionComponent
	if comp, exists := world.GetComponent(playerID, components.Position); exists {
		position = comp.(*components.PositionComponent)

		// Draw a separator
		for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
			s.tileset.DrawTile(screen, '-', x, 14, color.RGBA{180, 180, 180, 255})
		}

		// Display position information
		s.tileset.DrawString(screen, "LOCATION", config.GameScreenWidth+2, 16, color.RGBA{255, 230, 150, 255})
		s.tileset.DrawString(screen,
			"Pos: "+strconv.Itoa(position.X)+","+strconv.Itoa(position.Y),
			config.GameScreenWidth+2, 18, color.RGBA{200, 200, 255, 255})
		// Removed chunk coordinate display
	}

	// Draw game controls reminder at the bottom of the stats panel
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, config.GameScreenHeight-5, color.RGBA{180, 180, 180, 255})
	}
	s.tileset.DrawString(screen, "CONTROLS", config.GameScreenWidth+2, config.GameScreenHeight-4, color.RGBA{255, 230, 150, 255})
	s.tileset.DrawString(screen, "Arrow Keys: Move", config.GameScreenWidth+2, config.GameScreenHeight-2, color.RGBA{200, 200, 200, 255})
}

// drawMessagesPanel draws the message log panel
func (s *RenderSystem) drawMessagesPanel(screen *ebiten.Image) {
	// Draw messages panel border
	for x := 0; x < config.ScreenWidth; x++ {
		s.tileset.DrawTile(screen, '-', x, config.GameScreenHeight, color.RGBA{200, 200, 200, 255})
	}

	// Get message log
	messageLog := GetMessageLog()

	// Calculate how many messages can fit
	messagesAreaHeight := config.ScreenHeight - config.GameScreenHeight - 1
	maxMessages := messagesAreaHeight

	// Draw title for the message area
	s.tileset.DrawString(screen, "MESSAGE LOG", 1, config.GameScreenHeight+1, color.RGBA{255, 230, 150, 255})

	// Draw messages from the log (starting at line 2 to leave room for the title)
	messages := messageLog.RecentMessages(maxMessages)
	for i, msg := range messages {
		// Color coding based on message content (optional)
		msgColor := color.RGBA{200, 200, 200, 255} // Default gray

		// Use white for important messages (if they contain certain keywords)
		if len(msg) > 6 && (msg[:5] == "ERROR" || msg[:7] == "WARNING") {
			msgColor = color.RGBA{255, 100, 100, 255} // Red for errors/warnings
		}

		s.tileset.DrawString(screen, msg, 1, config.GameScreenHeight+2+i, msgColor)
	}
}
