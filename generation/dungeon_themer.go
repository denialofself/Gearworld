package generation

import (
	"math/rand"

	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/spawners"
)

// DungeonConfiguration defines a complete configuration for a dungeon
type DungeonConfiguration struct {
	Level                 int          // Dungeon depth/level
	Theme                 DungeonTheme // Theme of the dungeon
	Size                  DungeonSize  // Size of the dungeon
	DensityFactor         float64      // Monster density (1.0 = standard)
	HigherLevelChance     float64      // Chance of spawning higher level monsters (0.0-1.0)
	EvenHigherLevelChance float64      // Chance of spawning even higher level monsters (0.0-1.0)
}

// DungeonSize defines the size category of a dungeon
type DungeonSize int

const (
	SizeSmall  DungeonSize = iota // Small dungeon (one screen)
	SizeNormal                    // Normal dungeon (2-3 screens)
	SizeLarge                     // Large dungeon (5+ screens)
	SizeHuge                      // Huge dungeon (10+ screens)
)

// DungeonThemer handles creation of complete dungeons with themes
type DungeonThemer struct {
	world           *ecs.World
	dunGen          *DungeonGenerator
	populator       *DungeonPopulator
	mapGenerator    MapGenerator
	templateManager *data.EntityTemplateManager
	entitySpawner   *spawners.EntitySpawner
	rng             *rand.Rand
	logMessage      func(string) // Function for logging messages
}

// NewDungeonThemer creates a new dungeon theme manager
func NewDungeonThemer(world *ecs.World, mapGenerator MapGenerator, templateManager *data.EntityTemplateManager, entitySpawner *spawners.EntitySpawner, logFunc func(string)) *DungeonThemer {
	return &DungeonThemer{
		world:           world,
		dunGen:          NewDungeonGenerator(),
		populator:       NewDungeonPopulator(world, entitySpawner, templateManager),
		mapGenerator:    mapGenerator,
		templateManager: templateManager,
		entitySpawner:   entitySpawner,
		rng:             rand.New(rand.NewSource(0)), // Will be seeded via SetSeed
		logMessage:      logFunc,
	}
}

// SetSeed sets a specific seed for reproducible generation
func (t *DungeonThemer) SetSeed(seed int64) {
	t.rng = rand.New(rand.NewSource(seed))
	t.dunGen.SetSeed(seed)
	t.populator.SetSeed(seed)
}

// GenerateThemedDungeon generates a complete dungeon with a theme and appropriate monsters
func (t *DungeonThemer) GenerateThemedDungeon(config DungeonConfiguration) *ecs.Entity {
	var mapEntity *ecs.Entity

	// Generate the appropriate dungeon type based on size
	width, height := t.getDungeonDimensions(config.Size)

	// Generate different types of dungeons based on theme and size
	switch config.Size {
	case SizeSmall:
		mapEntity = t.mapGenerator.GenerateSmallBSPDungeon(t.world, width, height)
	case SizeLarge:
		mapEntity = t.mapGenerator.GenerateLargeBSPDungeon(t.world, width, height)
	case SizeHuge:
		mapEntity = t.mapGenerator.GenerateLargeBSPDungeon(t.world, width, height)
	default: // SizeNormal
		mapEntity = t.mapGenerator.GenerateSmallBSPDungeon(t.world, width, height)
	}

	// Get the map component
	var mapComp *components.MapComponent
	if comp, exists := t.world.GetComponent(mapEntity.ID, components.MapComponentID); exists {
		mapComp = comp.(*components.MapComponent)
	} else {
		if t.logMessage != nil {
			t.logMessage("Error: Failed to get map component")
		}
		return mapEntity
	}

	// Apply visual theming to the map
	t.applyMapTheming(mapComp, config.Theme)

	// Populate the dungeon with monsters
	options := PopulationOptions{
		DungeonLevel:          config.Level,
		Theme:                 config.Theme,
		DensityFactor:         config.DensityFactor,
		HigherLevelChance:     config.HigherLevelChance,
		EvenHigherLevelChance: config.EvenHigherLevelChance,
		PreferredTags:         t.getThemeTags(config.Theme),
		ExcludeTags:           nil, // No excluded tags by default
	}

	t.populator.PopulateDungeon(mapComp, options)

	return mapEntity
}

// getDungeonDimensions returns the width and height for a dungeon of the given size
func (t *DungeonThemer) getDungeonDimensions(size DungeonSize) (width int, height int) {
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
