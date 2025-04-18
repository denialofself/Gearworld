package generation

import (
	"fmt"
	"math/rand"
	"time"

	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/spawners"
	"ebiten-rogue/systems"
)

// DungeonTheme defines a theme for dungeon population
type DungeonTheme string

const (
	ThemeStandard  DungeonTheme = "standard"  // Standard mixed monster distribution
	ThemeUndead    DungeonTheme = "undead"    // Focus on undead monsters
	ThemeGoblinoid DungeonTheme = "goblinoid" // Focus on goblinoid creatures
	ThemeInsects   DungeonTheme = "insect"    // Focus on insect creatures
	ThemeDemonic   DungeonTheme = "demonic"   // Focus on demonic creatures
	ThemeAbandoned DungeonTheme = "abandoned" // Focus on abandoned structures
)

// DungeonPopulator handles spawning entities into dungeons based on difficulty and theme
type DungeonPopulator struct {
	world           *ecs.World
	entitySpawner   *spawners.EntitySpawner
	templateManager *data.EntityTemplateManager
	rng             *rand.Rand
	logMessage      func(string) // Function for logging messages
}

// PopulationOptions defines options for populating a dungeon
type PopulationOptions struct {
	DungeonLevel          int      // Dungeon depth/level (affects monster difficulty)
	DensityFactor         float64  // How many monsters per room (1.0 = standard)
	HigherLevelChance     float64  // Chance of spawning monsters from next level (0.0-1.0)
	EvenHigherLevelChance float64  // Chance of spawning monsters from two levels higher (0.0-1.0)
	PreferredTags         []string // Tags to prefer when choosing monsters
	ExcludeTags           []string // Tags to avoid when choosing monsters
}

// NewDungeonPopulator creates a new dungeon populator
func NewDungeonPopulator(world *ecs.World, entitySpawner *spawners.EntitySpawner, templateManager *data.EntityTemplateManager, logFunc func(string)) *DungeonPopulator {
	return &DungeonPopulator{
		world:           world,
		entitySpawner:   entitySpawner,
		templateManager: templateManager,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		logMessage:      logFunc,
	}
}

// SetSeed allows setting a specific seed for reproducible generation
func (p *DungeonPopulator) SetSeed(seed int64) {
	p.rng = rand.New(rand.NewSource(seed))
}

// PopulateDungeon adds monsters and items to the dungeon based on the given options
func (p *DungeonPopulator) PopulateDungeon(mapComp *components.MapComponent, mapEntityID ecs.EntityID, options PopulationOptions) {
	p.entitySpawner.SetSpawnMapID(mapEntityID)
	systems.GetDebugLog().Add(fmt.Sprintf("Populating dungeon with map ID %d", mapEntityID))

	// Count floor tiles for debugging
	floorTiles := 0
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				floorTiles++
			}
		}
	}
	systems.GetDebugLog().Add(fmt.Sprintf("Map has %d floor tiles", floorTiles))

	// Count rooms to estimate how many monsters to place
	roomCount := p.countRooms(mapComp)
	systems.GetDebugLog().Add(fmt.Sprintf("Found %d rooms in dungeon", roomCount))

	// Determine number of monsters based on room count and density factor
	monsterCount := int(float64(roomCount) * options.DensityFactor)
	if monsterCount < 1 && roomCount > 0 && options.DensityFactor > 0 {
		monsterCount = 1 // Ensure at least one monster if we have rooms and non-zero density
	}
	systems.GetDebugLog().Add(fmt.Sprintf("Placing %d monsters (rooms: %d * density: %.2f)", monsterCount, roomCount, options.DensityFactor))

	// Get eligible monster templates based on theme and level
	eligibleTemplates := p.getEligibleMonsterTemplates(options)
	systems.GetDebugLog().Add(fmt.Sprintf("Found %d eligible monster templates", len(eligibleTemplates)))
	for _, t := range eligibleTemplates {
		systems.GetDebugLog().Add(fmt.Sprintf("- Eligible monster: %s (level %d, tags: %v)", t.ID, t.Level, t.Tags))
	}

	// Place monsters throughout the dungeon
	monstersPlaced := 0
	for i := 0; i < monsterCount; i++ {
		// Find an empty position
		x, y := p.findEmptyPosition(mapComp)
		if x == -1 || y == -1 {
			systems.GetDebugLog().Add("No more empty positions found for monsters")
			break
		}

		// Select a monster template
		template := p.selectMonsterTemplate(eligibleTemplates, options)
		if template == nil {
			systems.GetDebugLog().Add("No valid monster template found")
			continue
		}

		// Create the monster
		_, err := p.entitySpawner.CreateEnemy(x, y, template.ID)
		if err != nil {
			systems.GetDebugLog().Add(fmt.Sprintf("Failed to create monster at %d,%d: %v", x, y, err))
			continue
		}
		monstersPlaced++
		systems.GetDebugLog().Add(fmt.Sprintf("Created monster %s at %d,%d (%d/%d)", template.ID, x, y, monstersPlaced, monsterCount))
	}
	systems.GetDebugLog().Add(fmt.Sprintf("Finished populating dungeon. Placed %d/%d monsters", monstersPlaced, monsterCount))
}

// countRooms counts the number of distinct rooms in the dungeon
func (p *DungeonPopulator) countRooms(mapComp *components.MapComponent) int {
	// Initialize visited grid
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	roomCount := 0
	totalFloorTiles := 0

	// Scan the map for unvisited floor tiles
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor && !visited[y][x] {
				// Found a new room, perform flood fill and count tiles
				roomTiles := p.floodFillAndCount(mapComp, x, y, visited)
				totalFloorTiles += roomTiles

				// For large areas, count them as multiple rooms based on size
				if roomTiles >= 9 {
					// Each 100 tiles counts as a room, with a minimum of 1 room
					roomsInArea := roomTiles / 100
					if roomsInArea < 1 {
						roomsInArea = 1
					}
					roomCount += roomsInArea
					systems.GetDebugLog().Add(fmt.Sprintf("Found area with %d floor tiles at (%d,%d), counting as %d rooms", roomTiles, x, y, roomsInArea))
				}
			}
		}
	}

	systems.GetDebugLog().Add(fmt.Sprintf("Total floor tiles: %d, Found %d rooms", totalFloorTiles, roomCount))

	// If we found no rooms but have floor tiles, count it as one room
	if roomCount == 0 && totalFloorTiles > 0 {
		roomCount = 1
		systems.GetDebugLog().Add("No distinct rooms found, but have floor tiles. Treating as one room.")
	}

	return roomCount
}

// floodFillAndCount marks all connected floor tiles as visited and returns the count
func (p *DungeonPopulator) floodFillAndCount(mapComp *components.MapComponent, x, y int, visited [][]bool) int {
	// Check bounds
	if x < 0 || x >= mapComp.Width || y < 0 || y >= mapComp.Height {
		return 0
	}

	// Check if already visited or not a floor tile
	if visited[y][x] || mapComp.Tiles[y][x] != components.TileFloor {
		return 0
	}

	// Mark as visited
	visited[y][x] = true
	count := 1

	// Recursively visit neighbors
	count += p.floodFillAndCount(mapComp, x-1, y, visited)
	count += p.floodFillAndCount(mapComp, x+1, y, visited)
	count += p.floodFillAndCount(mapComp, x, y-1, visited)
	count += p.floodFillAndCount(mapComp, x, y+1, visited)

	return count
}

// findEmptyPosition finds an empty floor tile in the map
func (p *DungeonPopulator) findEmptyPosition(mapComp *components.MapComponent) (int, int) {
	// Try to find a good spot (floor tile)
	for attempts := 0; attempts < 100; attempts++ {
		x := p.rng.Intn(mapComp.Width)
		y := p.rng.Intn(mapComp.Height)

		if p.isValidMonsterPosition(mapComp, x, y) {
			return x, y
		}
	}

	// Fallback: scan the map systematically
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if p.isValidMonsterPosition(mapComp, x, y) {
				return x, y
			}
		}
	}

	// No empty position found
	return -1, -1
}

// isValidMonsterPosition checks if a position is valid for monster placement
func (p *DungeonPopulator) isValidMonsterPosition(mapComp *components.MapComponent, x, y int) bool {
	// Check if position is within bounds
	if x < 0 || x >= mapComp.Width || y < 0 || y >= mapComp.Height {
		return false
	}

	// Check if position is a floor tile
	if mapComp.Tiles[y][x] != components.TileFloor {
		return false
	}

	// Check if position is already occupied by an entity
	entities := p.world.GetEntitiesWithComponent(components.Position)
	for _, entity := range entities {
		posComp, _ := p.world.GetComponent(entity.ID, components.Position)
		pos := posComp.(*components.PositionComponent)
		if pos.X == x && pos.Y == y {
			return false
		}
	}

	return true
}

// monsterTemplateInfo stores a monster template with its spawn weight
type monsterTemplateInfo struct {
	ID     string
	Weight int
	Level  int
}

// getEligibleMonsters returns weighted templates eligible for the dungeon
func (p *DungeonPopulator) getEligibleMonsters(options PopulationOptions) []monsterTemplateInfo {
	var eligibleMonsters []monsterTemplateInfo

	// Get all monster templates
	for id, template := range p.templateManager.Templates {
		// Skip templates that don't have the "enemy" tag
		isEnemy := false
		for _, tag := range template.Tags {
			if tag == "enemy" {
				isEnemy = true
				break
			}
		}
		if !isEnemy {
			continue
		}

		// Check if monster level is appropriate for this dungeon
		if template.Level > options.DungeonLevel+2 {
			continue // Monster is too high level
		}

		// Check if monster matches theme constraints
		if !p.matchesTheme(template, options) {
			continue
		}

		// Check if monster has excluded tags
		if p.hasExcludedTags(template, options.ExcludeTags) {
			continue
		}

		// Determine base weight based on level difference
		weight := template.SpawnWeight
		levelDiff := options.DungeonLevel - template.Level

		// Adjust weight based on level difference
		switch {
		case levelDiff < 0: // Monster is higher level than dungeon
			if levelDiff == -1 {
				// Level+1 monster: use higher level chance
				if p.rng.Float64() > options.HigherLevelChance {
					continue // Skip based on chance
				}
				weight = weight / 2 // Reduce weight for higher level monsters
			} else if levelDiff == -2 {
				// Level+2 monster: use even higher level chance
				if p.rng.Float64() > options.EvenHigherLevelChance {
					continue // Skip based on chance
				}
				weight = weight / 4 // Significantly reduce weight for much higher level monsters
			} else {
				continue // Monster is too high level
			}
		case levelDiff > 2: // Monster is much lower level than dungeon
			weight = weight / 4 // Significantly reduce weight for much lower level monsters
		case levelDiff > 0: // Monster is lower level than dungeon
			weight = weight / 2 // Somewhat reduce weight for lower level monsters
		}

		// Boost weight for monsters matching preferred tags
		if p.hasPreferredTags(template, options.PreferredTags) {
			weight = weight * 3 // Triple weight for preferred monsters
		}

		// Add to eligible monsters
		eligibleMonsters = append(eligibleMonsters, monsterTemplateInfo{
			ID:     id,
			Weight: weight,
			Level:  template.Level,
		})
	}

	return eligibleMonsters
}

// matchesTheme checks if a monster matches the dungeon theme
func (p *DungeonPopulator) matchesTheme(template *data.EntityTemplate, options PopulationOptions) bool {
	// Monster must match at least one of the preferred tags
	return p.hasPreferredTags(template, options.PreferredTags)
}

// hasTag checks if a template has a specific tag
func (p *DungeonPopulator) hasTag(template *data.EntityTemplate, tag string) bool {
	for _, templateTag := range template.Tags {
		if templateTag == tag {
			return true
		}
	}
	return false
}

// hasExcludedTags checks if a template has any excluded tags
func (p *DungeonPopulator) hasExcludedTags(template *data.EntityTemplate, excludedTags []string) bool {
	for _, excludeTag := range excludedTags {
		if p.hasTag(template, excludeTag) {
			return true
		}
	}
	return false
}

// hasPreferredTags checks if a template has any preferred tags
func (p *DungeonPopulator) hasPreferredTags(template *data.EntityTemplate, preferredTags []string) bool {
	for _, preferredTag := range preferredTags {
		if p.hasTag(template, preferredTag) {
			return true
		}
	}
	return false
}

// chooseMonsterTemplate selects a monster template based on weighted probability
func (p *DungeonPopulator) chooseMonsterTemplate(templates []monsterTemplateInfo) string {
	if len(templates) == 0 {
		return ""
	}

	// Calculate total weight
	totalWeight := 0
	for _, template := range templates {
		totalWeight += template.Weight
	}

	if totalWeight <= 0 {
		// If no valid weights, choose randomly
		return templates[p.rng.Intn(len(templates))].ID
	}

	// Select based on weight
	roll := p.rng.Intn(totalWeight)
	currentWeight := 0

	for _, template := range templates {
		currentWeight += template.Weight
		if roll < currentWeight {
			return template.ID
		}
	}

	// Fallback (should never reach here)
	return templates[0].ID
}

// getEligibleMonsterTemplates returns a list of monster templates that match the given options
func (p *DungeonPopulator) getEligibleMonsterTemplates(options PopulationOptions) []*data.EntityTemplate {
	var templates []*data.EntityTemplate

	// Get all monster templates
	for _, template := range p.templateManager.Templates {
		// Skip templates that don't have the "enemy" tag
		isEnemy := false
		for _, tag := range template.Tags {
			if tag == "enemy" {
				isEnemy = true
				break
			}
		}
		if !isEnemy {
			continue
		}

		// Check if template level is appropriate
		if template.Level > options.DungeonLevel+2 {
			continue // Skip monsters that are too high level
		}

		// Check if template has any excluded tags
		if p.hasExcludedTags(template, options.ExcludeTags) {
			continue
		}

		// Check if template matches theme constraints
		if !p.matchesTheme(template, options) {
			continue
		}

		templates = append(templates, template)
	}

	systems.GetDebugLog().Add(fmt.Sprintf("Found %d eligible monster templates", len(templates)))
	for _, t := range templates {
		systems.GetDebugLog().Add(fmt.Sprintf("- %s (level %d, tags: %v)", t.ID, t.Level, t.Tags))
	}

	return templates
}

// selectMonsterTemplate chooses a monster template based on weighted probability
func (p *DungeonPopulator) selectMonsterTemplate(templates []*data.EntityTemplate, options PopulationOptions) *data.EntityTemplate {
	if len(templates) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, template := range templates {
		weight := template.SpawnWeight
		levelDiff := template.Level - options.DungeonLevel
		if levelDiff == 1 && p.rng.Float64() < options.HigherLevelChance {
			weight *= 2
		} else if levelDiff == 2 && p.rng.Float64() < options.EvenHigherLevelChance {
			weight *= 3
		}

		totalWeight += weight
	}

	// Select a template based on weight
	roll := p.rng.Intn(totalWeight)
	currentWeight := 0
	for _, template := range templates {
		weight := template.SpawnWeight
		levelDiff := template.Level - options.DungeonLevel
		if levelDiff == 1 && p.rng.Float64() < options.HigherLevelChance {
			weight *= 2
		} else if levelDiff == 2 && p.rng.Float64() < options.EvenHigherLevelChance {
			weight *= 3
		}

		currentWeight += weight
		if roll < currentWeight {
			return template
		}
	}

	return templates[0] // Fallback to first template
}
