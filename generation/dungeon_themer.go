package generation

import (
	"fmt"
	"math/rand"

	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/spawners"
)

type GeneratorType int

const (
	GeneratorBSP      GeneratorType = iota // Binary Space Partitioning generator
	GeneratorCellular                      // Cellular Automata generator
	GeneratorRandom                        // Simple random rooms with corridors
)

// DungeonConfiguration defines a complete configuration for a dungeon
type DungeonConfiguration struct {
	Level                 int           // Dungeon depth/level
	Theme                 DungeonTheme  // Theme of the dungeon (legacy enum approach)
	Size                  DungeonSize   // Size of the dungeon
	Generator             GeneratorType // Type of dungeon generator to use
	DensityFactor         float64       // Monster density (1.0 = standard)
	HigherLevelChance     float64       // Chance of spawning higher level monsters (0.0-1.0)
	EvenHigherLevelChance float64       // Chance of spawning even higher level monsters (0.0-1.0)
	AddStairsUp           bool          // Whether to add stairs up near the player's spawn point
	ThemeID               string        // Optional ID of a JSON theme definition to use instead of Theme enum
}

// DungeonSize defines the size category of a dungeon
type DungeonSize int

const (
	SizeSmall  DungeonSize = iota // Small dungeon (one screen)
	SizeNormal                    // Normal dungeon (2-3 screens)
	SizeLarge                     // Large dungeon (5+ screens)
	SizeHuge                      // Huge dungeon (10+ screens)
)

// DungeonGeneratorInterface defines the methods for any dungeon generator
type DungeonGeneratorInterface interface {
	// Generate creates a new dungeon layout in the provided map component
	Generate(mapComp *components.MapComponent, size DungeonSize) [][4]int

	// SetSeed sets a specific random seed for reproducible generation
	SetSeed(seed int64)
}

// DungeonThemer handles creation of complete dungeons with themes
type DungeonThemer struct {
	world           *ecs.World
	dungeonGen      *DungeonGenerator
	populator       *DungeonPopulator
	templateManager *data.EntityTemplateManager
	entitySpawner   *spawners.EntitySpawner
	themeManager    *DungeonThemeManager
	rng             *rand.Rand
	logMessage      func(string) // Function for logging messages
}

// NewDungeonThemer creates a new dungeon theme manager
func NewDungeonThemer(world *ecs.World, templateManager *data.EntityTemplateManager, entitySpawner *spawners.EntitySpawner, logFunc func(string)) *DungeonThemer {
	dungeonGen := NewDungeonGenerator()
	return &DungeonThemer{
		world:           world,
		dungeonGen:      dungeonGen,
		populator:       NewDungeonPopulator(world, entitySpawner, templateManager),
		templateManager: templateManager,
		entitySpawner:   entitySpawner,
		themeManager:    NewDungeonThemeManager(),
		rng:             rand.New(rand.NewSource(0)), // Will be seeded via SetSeed
		logMessage:      logFunc,
	}
}

// SetSeed sets a specific seed for reproducible generation
func (t *DungeonThemer) SetSeed(seed int64) {
	t.rng = rand.New(rand.NewSource(seed))
	t.dungeonGen.SetSeed(seed)
	t.populator.SetSeed(seed)
}

// LoadThemesFromDirectory loads dungeon themes from JSON files
func (t *DungeonThemer) LoadThemesFromDirectory(directory string) error {
	return t.themeManager.LoadThemesFromDirectory(directory)
}

// GenerateThemedDungeon generates a complete dungeon with a theme and appropriate monsters
func (t *DungeonThemer) GenerateThemedDungeon(config DungeonConfiguration) *ecs.Entity {
	// Create map entity
	mapEntity := t.world.CreateEntity()
	mapEntity.AddTag("map")
	t.world.TagEntity(mapEntity.ID, "map")

	// Generate the appropriate dungeon type based on size
	width, height := t.getDungeonDimensions(config.Size)

	// Create map component
	mapComp := components.NewMapComponent(width, height)
	t.world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Add map type component
	t.world.AddComponent(mapEntity.ID, components.MapType, components.NewMapTypeComponent("dungeon", config.Level))

	// Generate dungeon based on generator type and size
	var generatorName string
	var rooms [][4]int

	switch config.Generator {
	case GeneratorCellular:
		generatorName = "Cellular Automata"
		switch config.Size {
		case SizeSmall:
			t.dungeonGen.GenerateSmallCellularDungeon(mapComp)
		case SizeLarge:
			t.dungeonGen.GenerateLargeCellularDungeon(mapComp)
		case SizeHuge:
			t.dungeonGen.GenerateGiantCellularDungeon(mapComp)
		default: // SizeNormal
			t.dungeonGen.GenerateSmallCellularDungeon(mapComp)
		}
		// For cellular automata, find the rooms
		rooms = t.dungeonGen.findAllOpenAreas(mapComp)
	case GeneratorRandom:
		generatorName = "Random Rooms"
		// Use the existing simple rooms and corridors generator
		rooms = t.generateRandomRoomsAndCorridors(mapComp, config.Size)
	default: // GeneratorBSP
		generatorName = "BSP"
		switch config.Size {
		case SizeSmall:
			t.dungeonGen.GenerateSmallBSPDungeon(mapComp)
		case SizeLarge:
			t.dungeonGen.GenerateLargeBSPDungeon(mapComp)
		case SizeHuge:
			t.dungeonGen.GenerateGiantBSPDungeon(mapComp)
		default: // SizeNormal
			t.dungeonGen.GenerateSmallBSPDungeon(mapComp)
		}
		// For BSP dungeons, find the rooms
		rooms = t.findRoomsInBSPDungeon(mapComp)
	}

	// Log the map generation
	if t.logMessage != nil {
		var themeName string
		if config.ThemeID != "" {
			if theme := t.themeManager.GetTheme(config.ThemeID); theme != nil {
				themeName = theme.Name
			} else {
				themeName = string(config.Theme) // Fallback to enum theme
			}
		} else {
			themeName = string(config.Theme)
		}
		t.logMessage(fmt.Sprintf("Generated a %s dungeon using %s generator", themeName, generatorName))
	}

	// If configured, add stairs up near player spawn
	if config.AddStairsUp {
		t.addStairsUpNearPlayerSpawn(mapComp)
	}

	// Apply theme to the dungeon
	var usingJsonTheme bool = false
	if config.ThemeID != "" {
		// Use JSON theme definition if available
		if theme := t.themeManager.GetTheme(config.ThemeID); theme != nil {
			t.applyThemeDefinition(mapComp, theme, rooms)
			usingJsonTheme = true
		} else {
			// Fallback to legacy theme enum
			t.applyMapTheming(mapComp, config.Theme)
		}
	} else {
		// Use legacy theme enum
		t.applyMapTheming(mapComp, config.Theme)
	}

	// Add dungeon features (stairs, water pools, etc.) ONLY if not using JSON theme
	if !usingJsonTheme {
		t.dungeonGen.AddFeatures(mapComp, rooms)
	} else {
		// If using JSON theme, still add stairs down if not already added in applyThemeDefinition
		// but skip other features (vegetation, pools) as those are controlled by the theme
		if mapComp != nil {
			// Check if stairs down already exist
			var hasStairsDown bool = false
			for y := 0; y < mapComp.Height; y++ {
				for x := 0; x < mapComp.Width; x++ {
					if mapComp.Tiles[y][x] == components.TileStairsDown {
						hasStairsDown = true
						if t.logMessage != nil {
							t.logMessage(fmt.Sprintf("Found existing stairs down at (%d,%d)", x, y))
						}

						// Place the stairs and create a stairs entity
						mapComp.SetTile(x, y, components.TileStairsDown)

						// Create a stairs entity
						stairsEntity := t.world.CreateEntity()
						stairsEntity.AddTag("stairs")
						t.world.TagEntity(stairsEntity.ID, "stairs")

						// Add position component
						t.world.AddComponent(stairsEntity.ID, components.Position, &components.PositionComponent{
							X: x,
							Y: y,
						})

						// Add map context component
						t.world.AddComponent(stairsEntity.ID, components.MapContext, components.NewMapContextComponent(mapEntity.ID))

						if t.logMessage != nil {
							t.logMessage(fmt.Sprintf("Added stairs down at (%d,%d)", x, y))
						}

						break
					}
				}
				if hasStairsDown {
					break
				}
			}

			// If no stairs down, add them in the last room
			if !hasStairsDown {
				// Find the last room (typically the goal/boss room)
				var lastRoom [4]int
				if len(rooms) > 0 {
					lastRoom = rooms[len(rooms)-1]
				}

				var stairsX, stairsY int
				stairsPlaced := false

				// Try to find a good spot for stairs
				for attempts := 0; attempts < 20 && !stairsPlaced; attempts++ {
					testX := lastRoom[0] + t.rng.Intn(lastRoom[2])
					testY := lastRoom[1] + t.rng.Intn(lastRoom[3])

					if mapComp.Tiles[testY][testX] == components.TileFloor {
						stairsX, stairsY = testX, testY
						stairsPlaced = true
					}
				}

				// If we still can't find a good spot, use the room center
				if !stairsPlaced {
					stairsX = lastRoom[0] + lastRoom[2]/2
					stairsY = lastRoom[1] + lastRoom[3]/2

					// If it's a wall, find the nearest floor tile
					if mapComp.IsWall(stairsX, stairsY) {
						for radius := 1; radius < 5 && !stairsPlaced; radius++ {
							for dy := -radius; dy <= radius && !stairsPlaced; dy++ {
								for dx := -radius; dx <= radius && !stairsPlaced; dx++ {
									nx, ny := stairsX+dx, stairsY+dy
									if nx >= 0 && nx < mapComp.Width && ny >= 0 && ny < mapComp.Height &&
										mapComp.Tiles[ny][nx] == components.TileFloor {
										stairsX, stairsY = nx, ny
										stairsPlaced = true
									}
								}
							}
						}
					} else {
						stairsPlaced = true
					}
				}

				// Place the stairs if we found a good spot
				if stairsPlaced {
					mapComp.SetTile(stairsX, stairsY, components.TileStairsDown)
					if t.logMessage != nil {
						t.logMessage(fmt.Sprintf("Added backup stairs down at (%d,%d)", stairsX, stairsY))
					}

					// Create a stairs entity at this position and tag it
					stairsEntity := t.world.CreateEntity()
					stairsEntity.AddTag("stairs")
					t.world.TagEntity(stairsEntity.ID, "stairs")

					// Add position component
					t.world.AddComponent(stairsEntity.ID, components.Position, &components.PositionComponent{
						X: stairsX,
						Y: stairsY,
					})

					// Add map context component
					t.world.AddComponent(stairsEntity.ID, components.MapContext, components.NewMapContextComponent(mapEntity.ID))
				} else if t.logMessage != nil {
					t.logMessage("WARNING: Could not find a suitable location for stairs down")
				}
			}
		}
	}

	// Important: Set the map's ID in the entity spawner before populating with monsters
	// This ensures all monsters get the correct MapContext
	if t.logMessage != nil {
		t.logMessage(fmt.Sprintf("Setting spawn map ID to %d for monster creation", mapEntity.ID))
	}
	t.entitySpawner.SetSpawnMapID(mapEntity.ID)

	// Populate the dungeon with monsters
	var options PopulationOptions

	if config.ThemeID != "" {
		// Use JSON theme definition if available
		if theme := t.themeManager.GetTheme(config.ThemeID); theme != nil {
			options = PopulationOptions{
				DungeonLevel:          config.Level,
				Theme:                 config.Theme, // Keep legacy theme for compatibility
				DensityFactor:         theme.DensityFactor,
				HigherLevelChance:     theme.HigherLevelChance,
				EvenHigherLevelChance: theme.EvenHigherLevelChance,
				PreferredTags:         theme.Tags,
				ExcludeTags:           theme.ExcludeTags,
			}
		} else {
			// Fallback to configuration values
			options = PopulationOptions{
				DungeonLevel:          config.Level,
				Theme:                 config.Theme,
				DensityFactor:         config.DensityFactor,
				HigherLevelChance:     config.HigherLevelChance,
				EvenHigherLevelChance: config.EvenHigherLevelChance,
				PreferredTags:         t.getThemeTags(config.Theme),
				ExcludeTags:           nil,
			}
		}
	} else {
		// Use configuration values
		options = PopulationOptions{
			DungeonLevel:          config.Level,
			Theme:                 config.Theme,
			DensityFactor:         config.DensityFactor,
			HigherLevelChance:     config.HigherLevelChance,
			EvenHigherLevelChance: config.EvenHigherLevelChance,
			PreferredTags:         t.getThemeTags(config.Theme),
			ExcludeTags:           nil,
		}
	}

	// If we have a boss chance from a JSON theme, potentially add a boss
	if config.ThemeID != "" {
		if theme := t.themeManager.GetTheme(config.ThemeID); theme != nil && theme.BossChance > 0 {
			if t.rng.Float64() < theme.BossChance && len(theme.BossTypes) > 0 {
				t.addBossMonster(mapComp, theme.BossTypes)
			}
		}
	}

	t.populator.PopulateDungeon(mapComp, options)

	return mapEntity
}

// getDungeonDimensions returns the width and height for a dungeon of the given size
func (t *DungeonThemer) getDungeonDimensions(size DungeonSize) (width, height int) {
	switch size {
	case SizeSmall:
		return 40, 30
	case SizeLarge:
		return 100, 70
	case SizeHuge:
		return 160, 120
	default: // SizeNormal
		return 80, 50
	}
}

// getThemeTags returns tags that should be preferred for a given theme
func (t *DungeonThemer) getThemeTags(theme DungeonTheme) []string {
	switch theme {
	case ThemeUndead:
		return []string{"undead", "ghost", "skeleton", "zombie"}
	case ThemeGoblinoid:
		return []string{"goblinoid", "humanoid", "orc", "goblin"}
	case ThemeInsects:
		return []string{"insect", "vermin", "spider", "bug"}
	case ThemeDemonic:
		return []string{"demon", "devil", "fiend", "hellspawn"}
	default:
		return []string{} // No specific tags for standard
	}
}

// applyMapTheming applies visual changes to the map based on theme
func (t *DungeonThemer) applyMapTheming(mapComp *components.MapComponent, theme DungeonTheme) {
	// Apply theme-specific visual changes to the map
	switch theme {
	case ThemeUndead:
		// Add tombstones, bones, etc.
		t.addTombstones(mapComp)
	case ThemeDemonic:
		// Add lava pools, ritual circles
		t.addLavaPools(mapComp)
	case ThemeInsects:
		// Add webs, egg sacs
		t.addWebs(mapComp)
	case ThemeGoblinoid:
		// Add crude furniture, campfires
		t.addCampfires(mapComp)
	}
}

// Theme-specific map decorations (placeholder implementations)
func (t *DungeonThemer) addTombstones(mapComp *components.MapComponent) {
	// Replace some floor tiles with tombstone-like features
	// This would be implemented with actual tile types when available
	// For now, just a placeholder that does nothing
}

func (t *DungeonThemer) addLavaPools(mapComp *components.MapComponent) {
	// Add some lava pools to the map
	poolCount := mapComp.Width * mapComp.Height / 400 // Roughly one pool per 400 tiles

	for i := 0; i < poolCount; i++ {
		// Find a suitable location
		for attempts := 0; attempts < 50; attempts++ {
			x := t.rng.Intn(mapComp.Width-4) + 2
			y := t.rng.Intn(mapComp.Height-4) + 2

			// Only place on floor tiles
			if mapComp.Tiles[y][x] == components.TileFloor {
				// Create a small lava pool
				poolSize := 2 + t.rng.Intn(3) // 2-4 tiles across
				for py := y - poolSize/2; py <= y+poolSize/2; py++ {
					for px := x - poolSize/2; px <= x+poolSize/2; px++ {
						// Check bounds
						if px >= 0 && px < mapComp.Width && py >= 0 && py < mapComp.Height {
							// Only convert floor tiles and make pool irregular
							if mapComp.Tiles[py][px] == components.TileFloor && t.rng.Intn(100) < 70 {
								mapComp.SetTile(px, py, components.TileLava)
							}
						}
					}
				}
				break
			}
		}
	}
}

func (t *DungeonThemer) addWebs(mapComp *components.MapComponent) {
	// This would add web decorations when we have web tiles
	// For now, just a placeholder that does nothing
}

func (t *DungeonThemer) addCampfires(mapComp *components.MapComponent) {
	// This would add campfire decorations when we have campfire tiles
	// For now, just a placeholder that does nothing
}

// findPlayerSpawnLocation finds a suitable location for player spawning
func (t *DungeonThemer) findPlayerSpawnLocation(mapComp *components.MapComponent) (int, int) {
	// Return a random empty position in the map
	return t.findEmptyPosition(mapComp)
}

// addStairsUpNearPlayerSpawn adds stairs up to a dungeon map
func (t *DungeonThemer) addStairsUpNearPlayerSpawn(mapComp *components.MapComponent) {
	// Find a suitable empty position for stairs
	x, y := t.findPlayerSpawnLocation(mapComp)

	// Try to find another empty spot nearby
	for dx := -3; dx <= 3; dx++ {
		for dy := -3; dy <= 3; dy++ {
			nx, ny := x+dx, y+dy
			// Skip the exact spawn position
			if dx == 0 && dy == 0 {
				continue
			}

			// Check if position is valid and not a wall
			if nx >= 0 && nx < mapComp.Width && ny >= 0 && ny < mapComp.Height &&
				mapComp.Tiles[ny][nx] == components.TileFloor {
				// Found a spot, place stairs up
				mapComp.SetTile(nx, ny, components.TileStairsUp)
				t.logMessage(fmt.Sprintf("Added stairs up at (%d,%d)", nx, ny))

				// Since we don't have mapEntity.ID available here, just place the tile
				// and skip creating a dedicated stairs entity for now

				return
			}
		}
	}

	// If all else fails, just place stairs at the spawn location
	mapComp.SetTile(x, y, components.TileStairsUp)
	t.logMessage(fmt.Sprintf("WARNING: Had to place stairs up at player spawn (%d,%d)", x, y))
	// Skip creating stairs entity since we don't have mapEntity.ID
}

// findEmptyPosition finds an empty floor tile in the map
func (t *DungeonThemer) findEmptyPosition(mapComp *components.MapComponent) (int, int) {
	// Try to find a good spot (floor tile)
	for attempts := 0; attempts < 100; attempts++ {
		x := t.rng.Intn(mapComp.Width)
		y := t.rng.Intn(mapComp.Height)

		if mapComp.Tiles[y][x] == components.TileFloor {
			return x, y
		}
	}

	// Fallback: scan the map systematically
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				return x, y
			}
		}
	}

	// Last resort: return the center of the map
	return mapComp.Width / 2, mapComp.Height / 2
}

// GetDungeonThemeFromLevel returns a recommended theme for a dungeon level
func GetDungeonThemeFromLevel(level int, rng *rand.Rand) DungeonTheme {
	// Define some themes based on level ranges
	// This could be expanded or loaded from config
	switch {
	case level <= 3:
		// Beginner levels more likely to have goblinoids
		if rng.Float64() < 0.6 {
			return ThemeGoblinoid
		}
		return ThemeStandard

	case level <= 6:
		// Mid levels more likely to have undead
		roll := rng.Float64()
		if roll < 0.4 {
			return ThemeUndead
		} else if roll < 0.7 {
			return ThemeInsects
		}
		return ThemeStandard

	case level <= 10:
		// Higher levels more likely to have demons
		roll := rng.Float64()
		if roll < 0.5 {
			return ThemeDemonic
		} else if roll < 0.8 {
			return ThemeUndead
		}
		return ThemeStandard

	default:
		// Deep levels heavily demonic or mixed
		if rng.Float64() < 0.7 {
			return ThemeDemonic
		}
		return ThemeStandard
	}
}

// applyThemeDefinition applies visual changes based on a theme definition
func (t *DungeonThemer) applyThemeDefinition(mapComp *components.MapComponent, theme *DungeonThemeDefinition, rooms [][4]int) {
	// Check if stairs down already exist
	stairsDownExists := false
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileStairsDown {
				stairsDownExists = true
				if t.logMessage != nil {
					t.logMessage(fmt.Sprintf("Found existing stairs down at (%d,%d)", x, y))
				}
				break
			}
		}
		if stairsDownExists {
			break
		}
	}

	// Add stairs down to the next level in the last room if needed
	if !stairsDownExists && len(rooms) > 0 {
		lastRoom := rooms[len(rooms)-1]

		// Find a suitable floor tile in the room
		var stairsX, stairsY int
		stairsPlaced := false

		// Try several positions within the room
		for attempts := 0; attempts < 20 && !stairsPlaced; attempts++ {
			testX := lastRoom[0] + t.rng.Intn(lastRoom[2])
			testY := lastRoom[1] + t.rng.Intn(lastRoom[3])

			if mapComp.Tiles[testY][testX] == components.TileFloor {
				stairsX, stairsY = testX, testY
				stairsPlaced = true
			}
		}

		// If we couldn't find a floor tile after several attempts, use center of room
		if !stairsPlaced {
			stairsX = lastRoom[0] + lastRoom[2]/2
			stairsY = lastRoom[1] + lastRoom[3]/2

			// Make sure it's a floor tile
			if mapComp.IsWall(stairsX, stairsY) {
				// Find the nearest floor tile
				for r := 1; r < 5 && !stairsPlaced; r++ {
					for dy := -r; dy <= r && !stairsPlaced; dy++ {
						for dx := -r; dx <= r && !stairsPlaced; dx++ {
							nx, ny := stairsX+dx, stairsY+dy
							if nx >= lastRoom[0] && nx < lastRoom[0]+lastRoom[2] &&
								ny >= lastRoom[1] && ny < lastRoom[1]+lastRoom[3] &&
								mapComp.Tiles[ny][nx] == components.TileFloor {
								stairsX, stairsY = nx, ny
								stairsPlaced = true
							}
						}
					}
				}
			} else {
				stairsPlaced = true
			}
		}

		// Place the stairs if we found a valid position
		if stairsPlaced {
			mapComp.SetTile(stairsX, stairsY, components.TileStairsDown)
			if t.logMessage != nil {
				t.logMessage(fmt.Sprintf("Added stairs down in last room at (%d,%d)", stairsX, stairsY))
			}
		}
	}

	// Place features using our generic function

	// Water pools
	if theme.WaterChance > 0 {
		t.placeFeaturePools(mapComp, components.TileWater, theme.WaterChance)
	}

	// Lava pools
	if theme.LavaChance > 0 {
		t.placeFeaturePools(mapComp, components.TileLava, theme.LavaChance)
	}

	// Grass patches
	if theme.GrassChance > 0 {
		t.placeFeature(mapComp, components.TileGrass, theme.GrassChance, []int{components.TileFloor})
	}

	// Trees
	if theme.TreeChance > 0 {
		t.placeFeature(mapComp, components.TileTree, theme.TreeChance, []int{components.TileFloor, components.TileGrass})
	}

	// TODO: Add support for special tiles when more tile types are available
}

// placeFeature places a specific feature type on the map based on chance percentage
// targetTiles specifies which tile types can be replaced (empty means any non-wall tile)
func (t *DungeonThemer) placeFeature(mapComp *components.MapComponent, featureType int, chance float64, targetTiles []int) {
	// Calculate how many features to place based on map size and chance
	featureCount := int(float64(mapComp.Width*mapComp.Height) * chance / 100.0)

	// Place the features
	for i := 0; i < featureCount; i++ {
		x := t.rng.Intn(mapComp.Width)
		y := t.rng.Intn(mapComp.Height)

		// Check if current tile is valid for replacement
		currentTile := mapComp.Tiles[y][x]

		// Skip walls and special tiles
		if mapComp.IsWall(x, y) || currentTile == components.TileStairsUp ||
			currentTile == components.TileStairsDown || currentTile == components.TileDoor {
			continue
		}

		// If target tiles are specified, check if current tile is in the list
		if len(targetTiles) > 0 {
			canReplace := false
			for _, validTile := range targetTiles {
				if currentTile == validTile {
					canReplace = true
					break
				}
			}
			if !canReplace {
				continue
			}
		}

		// Place the feature
		mapComp.SetTile(x, y, featureType)
	}
}

// placeFeaturePools places pool-type features (water, lava) that should appear in clusters
func (t *DungeonThemer) placeFeaturePools(mapComp *components.MapComponent, featureType int, chance float64) {
	// Calculate how many pools to place based on map size and chance
	// Pools need a lower divisor since each pool consists of multiple tiles
	poolCount := int(float64(mapComp.Width*mapComp.Height) * chance / 400.0)

	for i := 0; i < poolCount; i++ {
		// Find an empty spot for the pool
		var poolX, poolY int
		for attempts := 0; attempts < 50; attempts++ {
			poolX = t.rng.Intn(mapComp.Width-5) + 2
			poolY = t.rng.Intn(mapComp.Height-5) + 2
			if !mapComp.IsWall(poolX, poolY) {
				break
			}
		}

		// Create a small pool (3x3 to 5x5)
		poolSize := 3 + t.rng.Intn(3)
		for y := poolY; y < poolY+poolSize && y < mapComp.Height-1; y++ {
			for x := poolX; x < poolX+poolSize && x < mapComp.Width-1; x++ {
				if !mapComp.IsWall(x, y) && t.rng.Intn(100) < 70 { // Make pools irregular
					mapComp.SetTile(x, y, featureType)
				}
			}
		}
	}
}

// addPool adds a water or lava pool to the dungeon
// DEPRECATED: Use placeFeaturePools instead
func (t *DungeonThemer) addPool(mapComp *components.MapComponent, tileType int) {
	// Find an empty spot for the pool
	var poolX, poolY int
	for attempts := 0; attempts < 50; attempts++ {
		poolX = t.rng.Intn(mapComp.Width-5) + 2
		poolY = t.rng.Intn(mapComp.Height-5) + 2
		if !mapComp.IsWall(poolX, poolY) {
			break
		}
	}

	// Create a small pool (3x3 to 5x5)
	poolSize := 3 + t.rng.Intn(3)
	for y := poolY; y < poolY+poolSize && y < mapComp.Height-1; y++ {
		for x := poolX; x < poolX+poolSize && x < mapComp.Width-1; x++ {
			if !mapComp.IsWall(x, y) && t.rng.Intn(100) < 70 { // Make pools irregular
				mapComp.SetTile(x, y, tileType)
			}
		}
	}
}

// generateRandomRoomsAndCorridors creates a simple dungeon with random rooms and corridors
func (t *DungeonThemer) generateRandomRoomsAndCorridors(mapComp *components.MapComponent, size DungeonSize) [][4]int {
	// Determine number of rooms based on dungeon size
	var numRooms int
	var minRoomSize, maxRoomSize int

	switch size {
	case SizeSmall:
		numRooms = 5 + t.rng.Intn(5) // 5-9 rooms
		minRoomSize = 5
		maxRoomSize = 10
	case SizeLarge:
		numRooms = 15 + t.rng.Intn(10) // 15-24 rooms
		minRoomSize = 7
		maxRoomSize = 15
	case SizeHuge:
		numRooms = 40 + t.rng.Intn(20) // 40-59 rooms
		minRoomSize = 10
		maxRoomSize = 20
	default:
		numRooms = 8 + t.rng.Intn(8) // 8-15 rooms
		minRoomSize = 6
		maxRoomSize = 12
	}

	var rooms [][4]int // Store rooms as [x, y, width, height]

	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	for i := 0; i < numRooms; i++ {
		// Random room size
		roomWidth := minRoomSize + t.rng.Intn(maxRoomSize-minRoomSize+1)
		roomHeight := minRoomSize + t.rng.Intn(maxRoomSize-minRoomSize+1)

		// Random room position (leaving space for walls)
		roomX := t.rng.Intn(mapComp.Width-roomWidth-1) + 1
		roomY := t.rng.Intn(mapComp.Height-roomHeight-1) + 1

		// Check for overlap with existing rooms - simple collision detection
		overlaps := false
		for _, room := range rooms {
			if !(roomX+roomWidth < room[0] || roomX > room[0]+room[2] ||
				roomY+roomHeight < room[1] || roomY > room[1]+room[3]) {
				overlaps = true
				break
			}
		}

		if overlaps {
			// Try again
			i--
			continue
		}

		// Store the room
		rooms = append(rooms, [4]int{roomX, roomY, roomWidth, roomHeight})

		// Create the room
		for y := roomY; y < roomY+roomHeight; y++ {
			for x := roomX; x < roomX+roomWidth; x++ {
				if x >= 0 && x < mapComp.Width && y >= 0 && y < mapComp.Height {
					mapComp.SetTile(x, y, components.TileFloor)
				}
			}
		}

		// If this isn't the first room, connect it to the previous room
		if i > 0 {
			// Get the center of the current room
			currentX := roomX + roomWidth/2
			currentY := roomY + roomHeight/2

			// Get the center of the previous room
			prevRoom := rooms[i-1]
			prevX := prevRoom[0] + prevRoom[2]/2
			prevY := prevRoom[1] + prevRoom[3]/2

			// Create corridor between rooms
			t.dungeonGen.CreateCorridor(mapComp, currentX, currentY, prevX, prevY)
		}
	}

	return rooms
}

// findRoomsInBSPDungeon finds all rooms in a BSP-generated dungeon
func (t *DungeonThemer) findRoomsInBSPDungeon(mapComp *components.MapComponent) [][4]int {
	var rooms [][4]int

	// We'll use a "visited" grid to track areas we've already processed
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	// Scan the map for floor tiles that haven't been visited
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			// Look for floor tiles that haven't been visited yet
			if (mapComp.Tiles[y][x] == components.TileFloor ||
				mapComp.Tiles[y][x] == components.TileGrass) && !visited[y][x] {
				// Found a new room, perform flood fill to find its extent
				room := t.floodFillRoom(mapComp, x, y, visited)

				// Add this room to our collection if it's a reasonable size
				if room[2] >= 4 && room[3] >= 4 {
					rooms = append(rooms, room)
				}
			}
		}
	}

	return rooms
}

// floodFillRoom identifies a room's dimensions using flood fill
func (t *DungeonThemer) floodFillRoom(mapComp *components.MapComponent, startX, startY int, visited [][]bool) [4]int {
	// Queue for BFS flood fill
	queue := [][2]int{{startX, startY}}
	visited[startY][startX] = true

	// Track the room's bounds
	minX, minY := startX, startY
	maxX, maxY := startX, startY

	// Directions for neighbor checks (4-way connectivity)
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	// Process the queue
	for len(queue) > 0 {
		// Get the next position
		pos := queue[0]
		queue = queue[1:]
		x, y := pos[0], pos[1]

		// Update bounds
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}

		// Check neighbors
		for _, dir := range dirs {
			nx, ny := x+dir[0], y+dir[1]

			// Check bounds
			if nx < 0 || nx >= mapComp.Width || ny < 0 || ny >= mapComp.Height {
				continue
			}

			// If not visited and is a walkable tile, add to queue
			if !visited[ny][nx] && t.isWalkableForRoomFinding(mapComp.Tiles[ny][nx]) {
				visited[ny][nx] = true
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}

	// Return room dimensions [x, y, width, height]
	return [4]int{minX, minY, maxX - minX + 1, maxY - minY + 1}
}

// isWalkableForRoomFinding checks if a tile can be considered part of a room
func (t *DungeonThemer) isWalkableForRoomFinding(tileType int) bool {
	return tileType == components.TileFloor ||
		tileType == components.TileGrass ||
		tileType == components.TileDoor
}

// addBossMonster adds a boss monster to the dungeon
func (t *DungeonThemer) addBossMonster(mapComp *components.MapComponent, bossTypes []string) {
	if len(bossTypes) == 0 {
		return
	}

	// Find a good location for the boss (ideally in a large room)
	x, y := t.findEmptyPosition(mapComp)

	// Choose a random boss type from the list
	bossType := bossTypes[t.rng.Intn(len(bossTypes))]

	// Spawn the boss
	_, err := t.entitySpawner.CreateEnemy(x, y, bossType)
	if err == nil && t.logMessage != nil {
		t.logMessage(fmt.Sprintf("Added boss monster '%s' at %d,%d", bossType, x, y))
	}
}
