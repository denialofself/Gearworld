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
	tileset             *Tileset
	cameraX             int          // Camera X position
	cameraY             int          // Camera Y position
	cameraTargetID      ecs.EntityID // Entity the camera is following
	debugWindowActive   bool         // Whether the debug window is currently displayed
	debugScrollOffset   int          // Current scroll position in the debug log
	showInventory       bool         // Whether to show inventory instead of stats panel
	itemViewMode        bool         // Whether we're viewing a specific item's details
	selectedItemIndex   int          // Index of the currently selected item
	initialized         bool         // Whether the system has been initialized
	world               *ecs.World
	messageScrollOffset int // New field for message scrolling
}

// NewRenderSystem creates a new rendering system
func NewRenderSystem(tileset *Tileset) *RenderSystem {
	return &RenderSystem{
		tileset:           tileset,
		cameraX:           0,
		cameraY:           0,
		cameraTargetID:    0,
		debugWindowActive: false,
		debugScrollOffset: 0,
		showInventory:     false,
		itemViewMode:      false,
		selectedItemIndex: -1,
		initialized:       false,
	}
}

// Initialize sets up the render system
func (s *RenderSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	// Register to listen for camera update events
	world.GetEventManager().Subscribe(EventCameraUpdate, func(event ecs.Event) {
		cameraEvent := event.(CameraUpdateEvent)
		s.cameraX = cameraEvent.X
		s.cameraY = cameraEvent.Y
		s.cameraTargetID = cameraEvent.TargetID
	})

	// Register to listen for inventory UI events
	world.GetEventManager().Subscribe(EventInventoryUI, func(event ecs.Event) {
		uiEvent := event.(InventoryUIEvent)
		switch uiEvent.Action {
		case "open":
			s.showInventory = true
			s.itemViewMode = false
		case "close":
			s.showInventory = false
			s.itemViewMode = false
		case "select_item":
			s.selectedItemIndex = uiEvent.ItemIndex
		case "view_details":
			s.itemViewMode = true
			s.selectedItemIndex = uiEvent.ItemIndex
		}
	})

	// Register to listen for equipment change events - just for debug logging
	world.RegisterEventListener(s.handleEquipmentChange)

	s.initialized = true
	s.world = world
}

// Update performs any rendering-related updates
func (s *RenderSystem) Update(world *ecs.World, dt float64) {
	if !s.initialized {
		s.Initialize(world)
	}
}

// ToggleDebugWindow toggles the visibility of the debug message window
func (s *RenderSystem) ToggleDebugWindow() {
	s.debugWindowActive = !s.debugWindowActive
	if s.debugWindowActive {
		GetMessageLog().Add("Debug window activated")
	}
}

// ToggleInventoryDisplay toggles between stats panel and inventory panel
func (s *RenderSystem) ToggleInventoryDisplay() {
	s.showInventory = !s.showInventory
	// Reset item view mode when toggling inventory
	s.itemViewMode = false
	s.selectedItemIndex = -1
	if s.showInventory {
		GetMessageLog().Add("Inventory opened")
	} else {
		GetMessageLog().Add("Inventory closed")
	}
}

// IsInventoryOpen returns whether the inventory panel is currently shown
func (s *RenderSystem) IsInventoryOpen() bool {
	return s.showInventory
}

// IsItemViewMode returns whether we're currently viewing an item's details
func (s *RenderSystem) IsItemViewMode() bool {
	return s.itemViewMode
}

// ViewItemDetails sets up to view the details of a specific item
func (s *RenderSystem) ViewItemDetails(itemIndex int) {
	s.itemViewMode = true
	s.selectedItemIndex = itemIndex
}

// ExitItemView returns to the normal inventory view
func (s *RenderSystem) ExitItemView() {
	s.itemViewMode = false
	s.selectedItemIndex = -1
}

// No need for equipment caching - it will be rendered directly in drawStatsPanel

// Draw renders all entities with position and renderable components
func (s *RenderSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Clear the screen
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Check if we're in world map tester mode
	isWorldMapTester := len(world.GetEntitiesWithTag("worldmap_tester")) > 0

	// Draw the game area (map)
	s.drawGameScreen(world, screen)

	// Only draw UI elements if not in world map tester mode
	if !isWorldMapTester {
		if s.showInventory {
			s.drawInventoryPanel(world, screen)
		} else {
			s.drawStatsPanel(world, screen)
		}
		s.drawMessagesPanel(screen)
	}
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

	// Check if this is a world map (no FOV restrictions)
	var isWorldMap bool = false
	var isWorldMapTester bool = false
	if comp, exists := world.GetComponent(mapID, components.MapType); exists {
		mapTypeComp := comp.(*components.MapTypeComponent)
		isWorldMap = mapTypeComp.MapType == "worldmap"
		// Check if we're in the world map tester by looking for the "worldmap_tester" tag
		testerEntities := world.GetEntitiesWithTag("worldmap_tester")
		isWorldMapTester = len(testerEntities) > 0
	}

	// Get screen dimensions - use full screen only for world map tester
	var screenWidth, screenHeight int
	if isWorldMap && isWorldMapTester {
		screenWidth = screen.Bounds().Dx() / s.tileset.TileSize
		screenHeight = screen.Bounds().Dy() / s.tileset.TileSize
	} else {
		screenWidth = config.GameScreenWidth
		screenHeight = config.GameScreenHeight
	}

	// Draw map tiles that are visible in the viewport
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			// Convert screen position to world position
			// The camera position is already centered by the camera system
			worldX := x + cameraX
			worldY := y + cameraY

			// Skip if out of map bounds
			if worldX < 0 || worldX >= mapData.Width || worldY < 0 || worldY >= mapData.Height {
				continue
			}

			// Check tile visibility - on world maps everything is visible
			isVisible := mapData.Visible[worldY][worldX] || isWorldMap
			isExplored := mapData.Explored[worldY][worldX] || isWorldMap

			// Only draw tiles that are visible or have been explored
			if !isVisible && !isExplored {
				// Draw unexplored tiles as black
				s.tileset.DrawTile(screen, ' ', x, y, color.RGBA{0, 0, 0, 255})
				continue
			}

			// Get tile type at this world position
			tileType := mapData.Tiles[worldY][worldX]

			// Get the tile's visual definition from the mapping
			tileDef := tileMapping.GetTileDefinition(tileType)

			// Create a modified color based on visibility
			var fg color.Color

			if isVisible || isWorldMap {
				// Fully visible - use normal colors
				fg = tileDef.FG
			} else if isExplored {
				// Explored but not visible - darken the colors
				if fgRGBA, ok := tileDef.FG.(color.RGBA); ok {
					// Reduce brightness by 60%
					fg = color.RGBA{
						R: uint8(float64(fgRGBA.R) * 0.4),
						G: uint8(float64(fgRGBA.G) * 0.4),
						B: uint8(float64(fgRGBA.B) * 0.4),
						A: fgRGBA.A,
					}
				} else {
					// Default darkening if color conversion fails
					fg = color.RGBA{40, 40, 40, 255}
				}
			}

			// Draw the tile using either position or glyph based on the definition
			if tileDef.UseTilePos {
				// Use position-based tile reference
				tileID := NewTileID(tileDef.TileX, tileDef.TileY)
				s.tileset.DrawTileByID(screen, tileID, x, y, fg, 0)
			} else {
				// Use character-based reference
				s.tileset.DrawTile(screen, tileDef.Glyph, x, y, fg)
			}
		}
	}
}

// drawEntities draws all visible entities
func (s *RenderSystem) drawEntities(world *ecs.World, screen *ebiten.Image, cameraX, cameraY int) {
	// Get active map
	activeMap := s.getActiveMap(world)
	if activeMap == nil {
		return
	}
	activeMapID := activeMap.ID

	// Get map component
	mapComp, exists := world.GetComponent(activeMapID, components.MapComponentID)
	if !exists {
		return
	}
	mapComponent := mapComp.(*components.MapComponent)

	// Get map type
	var activeMapType string
	if typeComp, exists := world.GetComponent(activeMapID, components.MapType); exists {
		mapTypeComp := typeComp.(*components.MapTypeComponent)
		activeMapType = mapTypeComp.MapType
	}

	entitiesRendered := 0

	// First, draw the player if we're on the world map
	if activeMapType == "worldmap" {
		for _, entity := range world.GetEntitiesWithTag("player") {
			// Get position and renderable components
			posComp, hasPos := world.GetComponent(entity.ID, components.Position)
			rendComp, hasRend := world.GetComponent(entity.ID, components.Renderable)

			if hasPos && hasRend {
				pos := posComp.(*components.PositionComponent)
				rend := rendComp.(*components.RenderableComponent)

				// Use camera system to convert world position to screen position
				screenX := pos.X - cameraX
				screenY := pos.Y - cameraY

				// Only draw if within the visible game screen
				if screenX >= 0 && screenX < config.GameScreenWidth &&
					screenY >= 0 && screenY < config.GameScreenHeight {

					// Get rotation if entity has a RotationComponent
					var rotation float64
					if rotComp, exists := world.GetComponent(entity.ID, components.Rotation); exists {
						rotation = rotComp.(*components.RotationComponent).Angle
					}

					// Use the train sprite for the player on world map
					tileID := NewTileID(4, 14) // Train sprite position
					s.tileset.DrawTileByID(screen, tileID, screenX, screenY, rend.FG, rotation)
					entitiesRendered++
				}
			}
		}
	}

	// Then draw all other entities
	for _, entity := range world.GetAllEntities() {
		// Skip map and tilemap entities since we handle those separately
		if entity.HasTag("map") || entity.HasTag("tilemap") {
			continue
		}

		// Skip player on world map since we already drew it
		if activeMapType == "worldmap" && entity.HasTag("player") {
			continue
		}

		// Check if entity has a map context component
		if !world.HasComponent(entity.ID, components.MapContextID) {
			continue
		}

		mapContextComp, _ := world.GetComponent(entity.ID, components.MapContextID)
		mapContext := mapContextComp.(*components.MapContextComponent)

		// Skip entities that don't belong to the active map
		if mapContext.MapID != activeMapID {
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
			rend := rendComp.(*components.RenderableComponent)

			// Check if the entity's position is within bounds
			if pos.X < 0 || pos.X >= mapComponent.Width || pos.Y < 0 || pos.Y >= mapComponent.Height {
				continue
			}

			// Check if the entity is in a visible tile
			// Player is always visible
			isVisible := mapComponent.Visible[pos.Y][pos.X] || entity.HasTag("player") || activeMapType == "worldmap"
			isExplored := mapComponent.Explored[pos.Y][pos.X] || activeMapType == "worldmap"

			// Treat certain tile types as always visible when explored
			var tileTypeVisible bool = false
			if isExplored && !isVisible {
				// Get tile type at this position
				tileType := mapComponent.Tiles[pos.Y][pos.X]
				// Doors and stairs should remain visible when explored
				tileTypeVisible = tileType == components.TileDoor ||
					tileType == components.TileStairsUp ||
					tileType == components.TileStairsDown
			}

			// Only draw if the tile is visible or it's explored and should remain visible
			// On world map, always draw entities
			if !isVisible && !(isExplored && (entity.HasTag("stairs") || entity.HasTag("door") || tileTypeVisible)) && activeMapType != "worldmap" {
				continue
			}

			// If the tile is only explored but not currently visible, draw with reduced brightness
			// No darkening on world map
			var entityColor color.Color
			if isVisible || activeMapType == "worldmap" {
				entityColor = rend.FG
			} else if isExplored {
				// Entity is in an explored but not currently visible tile
				if fgRGBA, ok := rend.FG.(color.RGBA); ok {
					// Reduce brightness by 60%
					entityColor = color.RGBA{
						R: uint8(float64(fgRGBA.R) * 0.4),
						G: uint8(float64(fgRGBA.G) * 0.4),
						B: uint8(float64(fgRGBA.B) * 0.4),
						A: fgRGBA.A,
					}
				} else {
					// Default darkening if color conversion fails
					entityColor = color.RGBA{40, 40, 40, 255}
				}
			}

			// Use camera system to convert world position to screen position
			var screenX, screenY int
			screenX = pos.X - cameraX
			screenY = pos.Y - cameraY

			// Only draw entities within the visible game screen
			if screenX >= 0 && screenX < config.GameScreenWidth &&
				screenY >= 0 && screenY < config.GameScreenHeight {
				// Get rotation if entity has a RotationComponent
				var rotation float64
				if rotComp, exists := world.GetComponent(entity.ID, components.Rotation); exists {
					rotation = rotComp.(*components.RotationComponent).Angle
				}

				// Draw the entity using either position or glyph based approach
				if rend.UseTilePos {
					// Use position-based reference
					tileID := NewTileID(rend.TileX, rend.TileY)
					s.tileset.DrawTileByID(screen, tileID, screenX, screenY, entityColor, rotation)
				} else {
					// Use character-based reference
					s.tileset.DrawTile(screen, rend.Char, screenX, screenY, entityColor)
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
		tileID := NewTileID(12, 13)
		for x := 0; x < filledWidth; x++ {
			s.tileset.DrawTileByID(screen, tileID, config.GameScreenWidth+2+x, 7, color.RGBA{200, 0, 0, 255}, 0)
		}
		// Draw the dark portion of the bar
		for x := filledWidth; x < healthBarWidth; x++ {
			s.tileset.DrawTileByID(screen, tileID, config.GameScreenWidth+2+x, 7, color.RGBA{100, 0, 0, 255}, 0)
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

	// Draw a separator
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 14, color.RGBA{180, 180, 180, 255})
	}

	// Draw status section
	s.tileset.DrawString(screen, "STATUS", config.GameScreenWidth+2, 16, color.RGBA{255, 230, 150, 255})

	// Get player's active effects
	if effectComp, exists := world.GetComponent(playerID, components.Effect); exists {
		if effects, ok := effectComp.(*components.EffectComponent); ok {
			if len(effects.Effects) == 0 {
				s.tileset.DrawString(screen, "No active effects", config.GameScreenWidth+2, 18, color.RGBA{200, 200, 200, 255})
			} else {
				y := 18
				for _, effect := range effects.Effects {
					effectDesc := s.formatGameEffect(effect)
					// Use red color for negative effects like bleeding
					effectColor := color.RGBA{200, 200, 200, 255}
					if effect.Operation == components.EffectOpSubtract {
						effectColor = color.RGBA{255, 100, 100, 255}
					}
					s.tileset.DrawString(screen, effectDesc, config.GameScreenWidth+2, y, effectColor)
					y++
				}
			}
		}
	}

	// Draw a separator
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 22, color.RGBA{180, 180, 180, 255})
	}

	// Draw equipped items section
	if world.HasComponent(playerID, components.Equipment) {
		// Display equipment title
		s.tileset.DrawString(screen, "EQUIPMENT", config.GameScreenWidth+2, 24, color.RGBA{255, 230, 150, 255})

		// Fixed display positions for each equipment slot
		fixedPositions := map[components.EquipmentSlot]int{
			components.SlotHead:      26,
			components.SlotBody:      27,
			components.SlotMainHand:  28,
			components.SlotOffHand:   29,
			components.SlotFeet:      30,
			components.SlotAccessory: 31,
		}

		slotNames := map[components.EquipmentSlot]string{
			components.SlotHead:      "Head",
			components.SlotBody:      "Body",
			components.SlotMainHand:  "Weapon",
			components.SlotOffHand:   "Shield",
			components.SlotFeet:      "Feet",
			components.SlotAccessory: "Accessory",
		}

		// Get the equipment component directly
		if equipComp, exists := world.GetComponent(playerID, components.Equipment); exists {
			equipment := equipComp.(*components.EquipmentComponent)

			// Display equipment for each slot in fixed positions
			for slot, name := range slotNames {
				itemID := equipment.GetEquippedItem(slot)
				itemName := "-empty-"
				itemColor := color.RGBA{150, 150, 150, 255}

				// Get item name if equipped
				if itemID != 0 {
					if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
						itemName = nameComp.(*components.NameComponent).Name
						itemColor = color.RGBA{220, 220, 255, 255}
					} else {
						itemName = fmt.Sprintf("Item #%d", itemID)
						itemColor = color.RGBA{220, 220, 255, 255}
					}
				}

				// Use fixed position for each slot instead of incremental yPos
				slotText := fmt.Sprintf("%s: %s", name, itemName)
				s.tileset.DrawString(screen, slotText, config.GameScreenWidth+2, fixedPositions[slot], itemColor)
			}
		}

		// Draw a separator after equipment section
		for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
			s.tileset.DrawTile(screen, '-', x, 33, color.RGBA{180, 180, 180, 255})
		}
	}

	// Draw location section below equipment
	s.tileset.DrawString(screen, "LOCATION", config.GameScreenWidth+2, 35, color.RGBA{255, 230, 150, 255})

	// Get current map type and level
	var mapType string = "Unknown"
	var mapLevel int = -1
	activeMap := s.getActiveMap(world)
	if activeMap != nil {
		if typeComp, exists := world.GetComponent(activeMap.ID, components.MapType); exists {
			mapTypeComp := typeComp.(*components.MapTypeComponent)
			mapType = mapTypeComp.MapType
			mapLevel = mapTypeComp.Level
		}
	}

	// Display map information
	if mapType == "worldmap" {
		s.tileset.DrawString(screen, "Surface", config.GameScreenWidth+2, 37, color.RGBA{200, 200, 255, 255})
	} else {
		s.tileset.DrawString(screen, fmt.Sprintf("Dungeon Level %d", mapLevel), config.GameScreenWidth+2, 37, color.RGBA{200, 200, 255, 255})
	}

	// Get player position
	var position *components.PositionComponent
	if comp, exists := world.GetComponent(playerID, components.Position); exists {
		position = comp.(*components.PositionComponent)
	}

	// Display coordinates if we have position
	if position != nil {
		s.tileset.DrawString(screen,
			"Pos: "+strconv.Itoa(position.X)+","+strconv.Itoa(position.Y),
			config.GameScreenWidth+2, 38, color.RGBA{200, 200, 255, 255})
	}

	// Draw a separator before controls section
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 40, color.RGBA{180, 180, 180, 255})
	}

	// Draw game controls reminder at the bottom of the stats panel
	s.tileset.DrawString(screen, "CONTROLS", config.GameScreenWidth+2, 42, color.RGBA{255, 230, 150, 255})
	s.tileset.DrawString(screen, "Arrow Keys: Move", config.GameScreenWidth+2, 43, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "I: Inventory", config.GameScreenWidth+2, 44, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "PgUp/PgDn: Scroll Log", config.GameScreenWidth+2, 45, color.RGBA{200, 200, 200, 255})
}

// drawInventoryPanel draws the player inventory panel
func (s *RenderSystem) drawInventoryPanel(world *ecs.World, screen *ebiten.Image) {
	// Calculate inventory panel width (not used directly but kept for code consistency with other panels)
	_ = config.ScreenWidth - config.GameScreenWidth

	// Draw inventory panel border and background
	for y := 0; y < config.GameScreenHeight; y++ {
		// Draw vertical border
		s.tileset.DrawTile(screen, '|', config.GameScreenWidth, y, color.RGBA{200, 200, 200, 255})

		// Draw background for better readability
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

	// Check if player has an inventory
	var inventory *components.InventoryComponent
	if comp, exists := world.GetComponent(playerID, components.Inventory); exists {
		inventory = comp.(*components.InventoryComponent)

		if s.itemViewMode {
			// Draw item details view
			s.drawItemDetailsView(world, screen, inventory)
		} else {
			// Draw inventory list view
			s.drawInventoryListView(world, screen, inventory)
		}
	} else {
		s.tileset.DrawString(screen, "No inventory", config.GameScreenWidth+2, 6, color.RGBA{200, 200, 200, 255})
	}
}

// drawInventoryListView draws the list of items in the inventory
func (s *RenderSystem) drawInventoryListView(world *ecs.World, screen *ebiten.Image, inventory *components.InventoryComponent) {
	// Draw panel title
	s.tileset.DrawString(screen, "INVENTORY", config.GameScreenWidth+2, 1, color.RGBA{255, 255, 255, 255})
	// Draw horizontal separator under title
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 2, color.RGBA{180, 180, 180, 255})
	}

	// Display inventory info
	s.tileset.DrawString(screen,
		fmt.Sprintf("Items: %d/%d", inventory.Size(), inventory.MaxCapacity),
		config.GameScreenWidth+2, 4, color.RGBA{255, 230, 150, 255})

	// If no item is selected yet and we have items, select the first one
	if s.selectedItemIndex == -1 && inventory.Size() > 0 {
		s.selectedItemIndex = 0
	}

	// Display items list
	if inventory.Size() == 0 {
		s.tileset.DrawString(screen, "No items", config.GameScreenWidth+2, 6, color.RGBA{200, 200, 200, 255})
	} else {
		// Display the items
		for i, itemID := range inventory.Items {
			if i >= 15 { // Increased limit since we're not showing descriptions
				s.tileset.DrawString(screen, "...", config.GameScreenWidth+2, 6+i, color.RGBA{200, 200, 200, 255})
				break
			}

			// Get item name if it has one
			itemName := fmt.Sprintf("Item #%d", itemID)
			if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
				itemName = nameComp.(*components.NameComponent).Name
			}

			// Display the item with a letter for selection
			itemLetter := string(rune('a' + i))

			// Choose color based on selection
			itemColor := color.RGBA{200, 200, 255, 255}
			if i == s.selectedItemIndex {
				// Highlight the selected item
				itemColor = color.RGBA{255, 255, 100, 255}
				// Draw a selection indicator
				arrowTileID := NewTileID(0, 1)
				s.tileset.DrawTileByID(screen, arrowTileID, config.GameScreenWidth+1, 6+i, itemColor, 0)
			}

			s.tileset.DrawString(screen,
				fmt.Sprintf("%s) %s", itemLetter, itemName),
				config.GameScreenWidth+2, 6+i, itemColor)
		}
	}

	// Draw controls at bottom of panel
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, config.GameScreenHeight-6, color.RGBA{180, 180, 180, 255})
	}
	s.tileset.DrawString(screen, "CONTROLS", config.GameScreenWidth+2, config.GameScreenHeight-5, color.RGBA{255, 230, 150, 255})
	s.tileset.DrawString(screen, "I/ESC: Close inventory", config.GameScreenWidth+2, config.GameScreenHeight-4, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "Up/Down: Navigate items", config.GameScreenWidth+2, config.GameScreenHeight-3, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "Enter: View details", config.GameScreenWidth+2, config.GameScreenHeight-2, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "E: Equip item, U: Use item", config.GameScreenWidth+2, config.GameScreenHeight-1, color.RGBA{200, 200, 200, 255})
}

// drawItemDetailsView draws the detailed view of a selected item
func (s *RenderSystem) drawItemDetailsView(world *ecs.World, screen *ebiten.Image, inventory *components.InventoryComponent) {
	// Make sure the selected index is valid
	if s.selectedItemIndex < 0 || s.selectedItemIndex >= inventory.Size() {
		s.ExitItemView()
		return
	}

	// Get the selected item ID
	itemID := inventory.Items[s.selectedItemIndex]

	// Get item name
	itemName := fmt.Sprintf("Item #%d", itemID)
	if nameComp, exists := world.GetComponent(itemID, components.Name); exists {
		itemName = nameComp.(*components.NameComponent).Name
	}

	// Draw panel title
	s.tileset.DrawString(screen, "ITEM DETAILS", config.GameScreenWidth+2, 1, color.RGBA{255, 255, 255, 255})
	// Draw horizontal separator under title
	for x := config.GameScreenWidth + 1; x < config.ScreenWidth-1; x++ {
		s.tileset.DrawTile(screen, '-', x, 2, color.RGBA{180, 180, 180, 255})
	}

	// Draw item name with letter
	itemLetter := string(rune('a' + s.selectedItemIndex))
	s.tileset.DrawString(screen,
		fmt.Sprintf("%s) %s", itemLetter, itemName),
		config.GameScreenWidth+2, 4, color.RGBA{255, 230, 150, 255})

	// Get item component
	var itemComp *components.ItemComponent
	var hasItemComp bool
	if comp, exists := world.GetComponent(itemID, components.Item); exists {
		itemComp = comp.(*components.ItemComponent)
		hasItemComp = true
	}

	if hasItemComp {
		// Draw item description
		y := 6
		if itemComp.Description != "" {
			// Wrap description at 25 characters
			maxLineWidth := 25
			description := itemComp.Description

			for len(description) > 0 {
				lineLen := len(description)
				if lineLen > maxLineWidth {
					lineLen = maxLineWidth
				}

				s.tileset.DrawString(screen,
					description[:lineLen],
					config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})

				description = description[lineLen:]
				y++
			}

			y += 1 // Add a blank line
		}

		// Draw item stats
		s.tileset.DrawString(screen, "Item Info:", config.GameScreenWidth+2, y, color.RGBA{255, 230, 150, 255})
		y += 1

		// Show item type with a user-friendly description
		typeDesc := ""
		switch itemComp.ItemType {
		case "weapon":
			typeDesc = "Weapon (equips to main hand)"
		case "armor":
			typeDesc = "Armor (equips to body)"
		case "helmet":
			typeDesc = "Helmet (equips to head)"
		case "shield":
			typeDesc = "Shield (equips to off hand)"
		case "boots":
			typeDesc = "Boots (equips to feet)"
		case "accessory":
			typeDesc = "Accessory (equips to accessory slot)"
		case "potion":
			typeDesc = "Potion (consumable item)"
		case "scroll":
			typeDesc = "Scroll (consumable item)"
		default:
			typeDesc = itemComp.ItemType
		}

		s.tileset.DrawString(screen,
			fmt.Sprintf("Type: %s", typeDesc),
			config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})
		y += 1

		s.tileset.DrawString(screen,
			fmt.Sprintf("Value: %d", itemComp.Value),
			config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})
		y += 1

		s.tileset.DrawString(screen,
			fmt.Sprintf("Weight: %d", itemComp.Weight),
			config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})
		y += 2

		// Display item effects if any
		if itemComp.Data != nil {
			s.tileset.DrawString(screen, "Effects:", config.GameScreenWidth+2, y, color.RGBA{255, 230, 150, 255})
			y += 1

			if effects, ok := itemComp.Data.([]components.GameEffect); ok {
				if len(effects) == 0 {
					s.tileset.DrawString(screen, "None", config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})
					y += 1
				} else {
					for _, effect := range effects {
						effectDesc := s.formatGameEffect(effect)
						s.tileset.DrawString(screen, effectDesc, config.GameScreenWidth+2, y, color.RGBA{200, 200, 200, 255})
						y += 1
					}
				}
			}
		}
	} else {
		s.tileset.DrawString(screen, "No item data available", config.GameScreenWidth+2, 6, color.RGBA{200, 200, 200, 255})
	}

	// Draw controls at bottom of panel
	s.tileset.DrawString(screen, "CONTROLS", config.GameScreenWidth+2, config.GameScreenHeight-5, color.RGBA{255, 230, 150, 255})
	s.tileset.DrawString(screen, "ESC: Return to inventory", config.GameScreenWidth+2, config.GameScreenHeight-4, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "E: Equip item", config.GameScreenWidth+2, config.GameScreenHeight-3, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "U: Use item", config.GameScreenWidth+2, config.GameScreenHeight-2, color.RGBA{200, 200, 200, 255})
	s.tileset.DrawString(screen, "Up/Down: Previous/Next item", config.GameScreenWidth+2, config.GameScreenHeight-1, color.RGBA{200, 200, 200, 255})
}

// formatGameEffect formats a game effect in a user-friendly way
func (s *RenderSystem) formatGameEffect(effect components.GameEffect) string {
	var effectDesc string
	switch effect.Type {
	case components.EffectTypeInstant:
		effectDesc = fmt.Sprintf("Instantly %s by %.1f", effect.Operation, effect.Value)
	case components.EffectTypeDuration:
		effectDesc = fmt.Sprintf("%s by %.1f for %d turns", effect.Operation, effect.Value, effect.Duration)
	case components.EffectTypePeriodic:
		effectDesc = fmt.Sprintf("%s by %.1f every %d turns", effect.Operation, effect.Value, effect.Duration)
	case components.EffectTypeConditional:
		effectDesc = fmt.Sprintf("When condition met: %s by %.1f", effect.Operation, effect.Value)
	}
	return effectDesc
}

// drawMessagesPanel draws the message log panel
func (s *RenderSystem) drawMessagesPanel(screen *ebiten.Image) {
	// Draw messages panel border
	for x := 0; x < config.ScreenWidth; x++ {
		s.tileset.DrawTile(screen, '-', x, config.MessageWindowStartY, color.RGBA{200, 200, 200, 255})
	}

	// Get message log
	messageLog := GetMessageLog()

	// Calculate how many messages can fit in the reduced space
	messagesAreaHeight := config.MessageWindowHeight - 1 // Leave room for title
	maxMessages := messagesAreaHeight

	// Draw title for the message area
	s.tileset.DrawString(screen, "MESSAGE LOG", 1, config.MessageWindowStartY+1, color.RGBA{255, 230, 150, 255})

	// Get visible messages based on scroll position
	messages := messageLog.RecentMessages(100) // Get all messages
	startIdx := s.messageScrollOffset
	if startIdx > len(messages)-maxMessages {
		startIdx = len(messages) - maxMessages
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Draw visible messages
	for i := 0; i < maxMessages && startIdx+i < len(messages); i++ {
		msg := messages[startIdx+i]
		msgColor := msg.GetColor()
		s.tileset.DrawString(screen, msg.Text, 1, config.MessageWindowStartY+2+i, msgColor)
	}

	// Draw scroll indicators if needed
	if len(messages) > maxMessages {
		if s.messageScrollOffset > 0 {
			s.tileset.DrawTile(screen, '▲', config.ScreenWidth-2, config.MessageWindowStartY+2, color.RGBA{200, 200, 200, 255})
		}
		if s.messageScrollOffset < len(messages)-maxMessages {
			s.tileset.DrawTile(screen, '▼', config.ScreenWidth-2, config.MessageWindowStartY+maxMessages, color.RGBA{200, 200, 200, 255})
		}
	}

	// Draw debug window if active
	if s.debugWindowActive {
		s.drawDebugWindow(screen)
	}
}

// ScrollMessagesUp scrolls the message window up one line
func (s *RenderSystem) ScrollMessagesUp() {
	if s.messageScrollOffset > 0 {
		s.messageScrollOffset--
	}
}

// ScrollMessagesDown scrolls the message window down one line
func (s *RenderSystem) ScrollMessagesDown() {
	messageLog := GetMessageLog()
	messages := messageLog.RecentMessages(100)
	maxVisible := config.MessageWindowHeight - 2 // Account for title and border

	if s.messageScrollOffset < len(messages)-maxVisible {
		s.messageScrollOffset++
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

	// Extract the visible messages as strings
	visibleMessages := make([]string, endIdx-startIdx)
	for i := 0; i < endIdx-startIdx; i++ {
		visibleMessages[i] = debugLog.Messages[startIdx+i].Text
	}

	return visibleMessages
}

// SelectNextItem selects the next item in the inventory
func (s *RenderSystem) SelectNextItem(world *ecs.World) {
	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}

	playerID := playerEntities[0].ID

	// Check if player has an inventory
	var inventory *components.InventoryComponent
	if comp, exists := world.GetComponent(playerID, components.Inventory); exists {
		inventory = comp.(*components.InventoryComponent)

		// If inventory is empty, do nothing
		if inventory.Size() == 0 {
			return
		}

		// Move to the next item, or wrap around
		s.selectedItemIndex++
		if s.selectedItemIndex >= inventory.Size() {
			s.selectedItemIndex = 0
		}
	}
}

// SelectPreviousItem selects the previous item in the inventory
func (s *RenderSystem) SelectPreviousItem(world *ecs.World) {
	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}

	playerID := playerEntities[0].ID

	// Check if player has an inventory
	var inventory *components.InventoryComponent
	if comp, exists := world.GetComponent(playerID, components.Inventory); exists {
		inventory = comp.(*components.InventoryComponent)

		// If inventory is empty, do nothing
		if inventory.Size() == 0 {
			return
		}

		// Move to the previous item, or wrap around
		s.selectedItemIndex--
		if s.selectedItemIndex < 0 {
			s.selectedItemIndex = inventory.Size() - 1
		}
	}
}

// GetSelectedItemIndex returns the currently selected item index
func (s *RenderSystem) GetSelectedItemIndex() int {
	return s.selectedItemIndex
}

// SetSelectedItemIndex sets the selected item index
func (s *RenderSystem) SetSelectedItemIndex(index int) {
	s.selectedItemIndex = index
}

// handleEquipmentChange listens for equipment change events
func (s *RenderSystem) handleEquipmentChange(world *ecs.World, event interface{}) {
	// Just log equipment changes for debugging
	switch evt := event.(type) {
	case ItemEquippedEvent:
		GetDebugLog().Add(fmt.Sprintf("Equipment change detected: Equipped item %d in slot %s", evt.ItemID, evt.Slot))
	case ItemUnequippedEvent:
		GetDebugLog().Add(fmt.Sprintf("Equipment change detected: Unequipped item %d from slot %s", evt.ItemID, evt.Slot))
	}
}

// Equipment rendering is now done directly in the drawStatsPanel method
// without any caching or intermediate updates

// getActiveMap returns the currently active map entity
func (s *RenderSystem) getActiveMap(world *ecs.World) *ecs.Entity {
	// Find the MapRegistrySystem
	for _, system := range world.GetSystems() {
		if mapRegistry, ok := system.(interface{ GetActiveMap() *ecs.Entity }); ok {
			// Check if this is the MapRegistrySystem by checking the type name
			if fmt.Sprintf("%T", system) == "*systems.MapRegistrySystem" {
				return mapRegistry.GetActiveMap()
			}
		}
	}
	return nil
}
