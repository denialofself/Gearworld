package systems

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
)

// RenderSystem handles drawing entities to the screen
type RenderSystem struct {
	tileset           *Tileset
	cameraSystem      *CameraSystem // Reference to the camera system
	debugWindowActive bool          // Whether the debug window is currently displayed
	debugScrollOffset int           // Current scroll position in the debug log
}

// NewRenderSystem creates a new rendering system
func NewRenderSystem(tileset *Tileset) *RenderSystem {
	return &RenderSystem{
		tileset:           tileset,
		cameraSystem:      nil, // Will be set via SetCameraSystem
		debugWindowActive: false,
		debugScrollOffset: 0,
	}
}

// SetCameraSystem sets the camera system to be used for rendering
func (s *RenderSystem) SetCameraSystem(cameraSystem *CameraSystem) {
	s.cameraSystem = cameraSystem
}

// ToggleDebugWindow toggles the visibility of the debug message window
func (s *RenderSystem) ToggleDebugWindow() {
	s.debugWindowActive = !s.debugWindowActive
	if s.debugWindowActive {
		GetMessageLog().Add("Debug window activated")
	}
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

	// Find the active map - always use MapRegistrySystem for consistency
	var activeMap *ecs.Entity
	var activeMapRegistrySystem interface {
		GetActiveMap() *ecs.Entity
	}

	// Find the MapRegistrySystem
	for _, system := range world.GetSystems() {
		if mapRegistry, ok := system.(interface{ GetActiveMap() *ecs.Entity }); ok {
			// Check if this is the MapRegistrySystem by checking the type name
			if fmt.Sprintf("%T", system) == "*systems.MapRegistrySystem" {
				activeMapRegistrySystem = mapRegistry
				break
			}
		}
	}

	// Get the active map from the registry system
	if activeMapRegistrySystem != nil {
		activeMap = activeMapRegistrySystem.GetActiveMap()
	}

	// If still no active map, log an error
	if activeMap == nil {
		GetDebugLog().Add("RenderSystem: No active map found")
		return
	}

	// Draw the active map
	s.drawStandardMap(world, screen, activeMap.ID, tileMapping, cameraX, cameraY)

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
	// Find the active map - always use MapRegistrySystem for consistency
	var activeMapID ecs.EntityID
	var activeMapRegistrySystem interface {
		GetActiveMap() *ecs.Entity
	}

	// Find the MapRegistrySystem
	for _, system := range world.GetSystems() {
		if mapRegistry, ok := system.(interface{ GetActiveMap() *ecs.Entity }); ok {
			// Check if this is the MapRegistrySystem by checking the type name
			if fmt.Sprintf("%T", system) == "*systems.MapRegistrySystem" {
				activeMapRegistrySystem = mapRegistry
				break
			}
		}
	}

	// Get the active map from the registry system
	if activeMapRegistrySystem != nil {
		if activeMap := activeMapRegistrySystem.GetActiveMap(); activeMap != nil {
			activeMapID = activeMap.ID
		}
	}

	// If no active map found, we can't properly filter entities
	if activeMapID == 0 {
		GetDebugLog().Add("RenderSystem: No active map found for entity rendering")
		return
	}

	// Get active map type for additional filtering
	var activeMapType string
	activeMapEntity := world.GetEntity(activeMapID)
	if activeMapEntity != nil && world.HasComponent(activeMapID, components.MapType) {
		mapTypeComp, _ := world.GetComponent(activeMapID, components.MapType)
		activeMapType = mapTypeComp.(*components.MapTypeComponent).MapType
	}

	// Track entities rendered for debugging
	entitiesRendered := 0

	// Get all entities to render
	for _, entity := range world.GetAllEntities() {
		// Skip map and tilemap entities since we handle those separately
		if entity.HasTag("map") || entity.HasTag("tilemap") {
			continue
		}

		// Check if entity has a map context component
		if world.HasComponent(entity.ID, components.MapContextID) {
			mapContextComp, _ := world.GetComponent(entity.ID, components.MapContextID)
			mapContext := mapContextComp.(*components.MapContextComponent)

			// Skip entities that don't belong to the active map
			if mapContext.MapID != activeMapID {
				// Debug logging for enemies on the wrong map context
				if entity.HasTag("enemy") && world.HasComponent(entity.ID, components.Name) {
					nameComp, _ := world.GetComponent(entity.ID, components.Name)
					name := nameComp.(*components.NameComponent)
					GetDebugLog().Add(fmt.Sprintf("DEBUG: Enemy '%s' (ID: %d) not rendered - wrong map context: %d vs active: %d",
						name.Name, entity.ID, mapContext.MapID, activeMapID))
				}
				continue
			}
		} else {
			// Debug log if entity doesn't have map context
			if entity.HasTag("ai") || entity.HasTag("enemy") {
				GetDebugLog().Add(fmt.Sprintf("Entity %d has no MapContext", entity.ID))
			}
			continue
		}

		// Extra safety - if we're on the world map, don't render enemies
		if activeMapType == "worldmap" && entity.HasTag("enemy") {
			continue
		}

		// Only render entities that have both Position and Renderable components
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		rendComp, hasRend := world.GetComponent(entity.ID, components.Renderable)

		if hasPos && hasRend {
			pos := posComp.(*components.PositionComponent)
			rend := rendComp.(*components.RenderableComponent) // Use camera system to convert world position to screen position
			var screenX, screenY int
			if s.cameraSystem != nil {
				screenX, screenY = s.cameraSystem.WorldToScreen(world, pos.X, pos.Y)
			} else {
				// Fallback if camera system is not available
				screenX = pos.X - cameraX
				screenY = pos.Y - cameraY
			}

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
				entitiesRendered++
			}
		}
	}

	// Debug log for number of entities rendered
	if activeMapType == "worldmap" {
		GetDebugLog().Add(fmt.Sprintf("DEBUG: Rendered %d entities on world map", entitiesRendered))
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

	// Draw debug window if active
	if s.debugWindowActive {
		s.drawDebugWindow(screen)
	}
}

// drawDebugWindow draws the debug message window overlay
func (s *RenderSystem) drawDebugWindow(screen *ebiten.Image) {
	// Define debug window dimensions
	windowWidth := config.ScreenWidth * 3 / 4
	windowHeight := config.ScreenHeight * 3 / 4
	startX := (config.ScreenWidth - windowWidth) / 2
	startY := (config.ScreenHeight - windowHeight) / 2

	// Instead of a semi-transparent overlay, draw a completely new background
	// First, fill the entire screen with a solid dark blue-gray color
	for y := 0; y < config.ScreenHeight; y++ {
		for x := 0; x < config.ScreenWidth; x++ {
			// Completely opaque dark background
			s.tileset.DrawTile(screen, ' ', x, y, color.RGBA{20, 20, 30, 255})
		}
	}

	// Create a subtle pattern in the background to make it look like a separate screen
	for y := 0; y < config.ScreenHeight; y += 4 {
		for x := 0; x < config.ScreenWidth; x += 8 {
			s.tileset.DrawTile(screen, '·', x, y, color.RGBA{40, 40, 60, 255})
		}
	}

	// Draw solid black window background that's clearly different from the patterned background
	for y := 0; y < windowHeight; y++ {
		for x := 0; x < windowWidth; x++ {
			// Solid black background for the actual window
			s.tileset.DrawTile(screen, ' ', startX+x, startY+y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Draw window border (white)
	borderColor := color.RGBA{255, 255, 255, 255}
	for x := 0; x < windowWidth; x++ {
		s.tileset.DrawTile(screen, '═', startX+x, startY, borderColor)
		s.tileset.DrawTile(screen, '═', startX+x, startY+windowHeight-1, borderColor)
	}
	for y := 0; y < windowHeight; y++ {
		s.tileset.DrawTile(screen, '║', startX, startY+y, borderColor)
		s.tileset.DrawTile(screen, '║', startX+windowWidth-1, startY+y, borderColor)
	}

	// Draw window corners (white)
	s.tileset.DrawTile(screen, '╔', startX, startY, borderColor)
	s.tileset.DrawTile(screen, '╗', startX+windowWidth-1, startY, borderColor)
	s.tileset.DrawTile(screen, '╚', startX, startY+windowHeight-1, borderColor)
	s.tileset.DrawTile(screen, '╝', startX+windowWidth-1, startY+windowHeight-1, borderColor)

	// Draw window title (white text)
	titleColor := color.RGBA{255, 255, 255, 255}
	s.tileset.DrawString(screen, "DEBUG MESSAGES (ESC to close, ↑/↓ to scroll)", startX+2, startY+1, titleColor)

	// Draw separator under title
	for x := 0; x < windowWidth-2; x++ {
		s.tileset.DrawTile(screen, '─', startX+1+x, startY+2, borderColor)
	}

	// Get debug messages
	debugLog := GetDebugLog()
	maxVisibleMessages := windowHeight - 6 // Account for borders, title, and scroll info

	// Implement scrolling
	totalMessages := len(debugLog.Messages)
	scrollOffset := s.getDebugScrollOffset(totalMessages, maxVisibleMessages)

	// Display visible messages with white text
	visibleMessages := s.getVisibleDebugMessages(debugLog, scrollOffset, maxVisibleMessages)
	messageColor := color.RGBA{255, 255, 255, 255}

	for i, msg := range visibleMessages {
		s.tileset.DrawString(screen, msg, startX+2, startY+3+i, messageColor)
	}

	// Draw scroll indicators if needed
	if totalMessages > maxVisibleMessages {
		if scrollOffset > 0 {
			s.tileset.DrawTile(screen, '▲', startX+windowWidth-3, startY+3, messageColor)
		}
		if scrollOffset < totalMessages-maxVisibleMessages {
			s.tileset.DrawTile(screen, '▼', startX+windowWidth-3, startY+windowHeight-3, messageColor)
		}

		// Draw scroll position indicator
		scrollInfo := fmt.Sprintf("%d-%d/%d", scrollOffset+1,
			min(scrollOffset+maxVisibleMessages, totalMessages),
			totalMessages)
		s.tileset.DrawString(screen, scrollInfo, startX+windowWidth-len(scrollInfo)-4, startY+windowHeight-2, messageColor)
	}
}

// IsDebugWindowActive returns whether the debug window is currently displayed
func (s *RenderSystem) IsDebugWindowActive() bool {
	return s.debugWindowActive
}

// ScrollDebugUp scrolls the debug window up one line
func (s *RenderSystem) ScrollDebugUp() {
	if s.debugScrollOffset > 0 {
		s.debugScrollOffset--
	}
}

// ScrollDebugDown scrolls the debug window down one line
func (s *RenderSystem) ScrollDebugDown() {
	debugLog := GetDebugLog()
	totalMessages := len(debugLog.Messages)
	maxVisibleMessages := config.ScreenHeight*3/4 - 6 // Same calculation as in drawDebugWindow

	if s.debugScrollOffset < totalMessages-maxVisibleMessages {
		s.debugScrollOffset++
	}
}

// getDebugScrollOffset returns the current scroll offset, ensuring it's in valid range
func (s *RenderSystem) getDebugScrollOffset(totalMessages, maxVisibleMessages int) int {
	// If there are fewer messages than can fit in the window, no scrolling needed
	if totalMessages <= maxVisibleMessages {
		return 0
	}

	// Ensure scroll offset is in valid range
	maxOffset := totalMessages - maxVisibleMessages
	if s.debugScrollOffset > maxOffset {
		s.debugScrollOffset = maxOffset
	}
	if s.debugScrollOffset < 0 {
		s.debugScrollOffset = 0
	}

	return s.debugScrollOffset
}

// getVisibleDebugMessages returns the slice of messages that should be visible
func (s *RenderSystem) getVisibleDebugMessages(debugLog *MessageLog, scrollOffset, maxVisible int) []string {
	if len(debugLog.Messages) == 0 {
		return []string{"No debug messages yet"}
	}

	// Calculate which messages to show based on scroll offset
	startIdx := len(debugLog.Messages) - maxVisible - scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(debugLog.Messages) {
		endIdx = len(debugLog.Messages)
	}

	// Extract the visible messages
	visibleMessages := make([]string, endIdx-startIdx)
	for i := 0; i < endIdx-startIdx; i++ {
		visibleMessages[i] = debugLog.Messages[startIdx+i]
	}

	return visibleMessages
}
