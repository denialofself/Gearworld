package generation

import (
	"fmt"
	"math/rand"
	"time"

	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/spawners"
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
}

// PopulationOptions defines options for populating a dungeon
type PopulationOptions struct {
	DungeonLevel          int          // Dungeon depth/level (affects monster difficulty)
	Theme                 DungeonTheme // Theme of the dungeon
	DensityFactor         float64      // How many monsters per room (1.0 = standard)
	HigherLevelChance     float64      // Chance of spawning monsters from next level (0.0-1.0)
	EvenHigherLevelChance float64      // Chance of spawning monsters from two levels higher (0.0-1.0)
	PreferredTags         []string     // Tags to prefer when choosing monsters
	ExcludeTags           []string     // Tags to avoid when choosing monsters
}

// NewDungeonPopulator creates a new dungeon populator
func NewDungeonPopulator(world *ecs.World, entitySpawner *spawners.EntitySpawner, templateManager *data.EntityTemplateManager) *DungeonPopulator {
	return &DungeonPopulator{
		world:           world,
		entitySpawner:   entitySpawner,
		templateManager: templateManager,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetSeed allows setting a specific seed for reproducible generation
func (p *DungeonPopulator) SetSeed(seed int64) {
	p.rng = rand.New(rand.NewSource(seed))
}

// PopulateDungeon adds monsters to a dungeon based on the provided options
func (p *DungeonPopulator) PopulateDungeon(mapComp *components.MapComponent, options PopulationOptions) {
	// Determine number of monsters based on map size and density factor
	roomCount := p.countRooms(mapComp)
	monstersPerRoom := 2.0 * options.DensityFactor // Base of 2 monsters per room on average
	totalMonsters := int(float64(roomCount) * monstersPerRoom)

	// Get suitable monster templates for this dungeon level and theme
	monsterTemplates := p.getEligibleMonsters(options)

	// Log monster template information for debugging
	if len(monsterTemplates) == 0 {
		fmt.Printf("Warning: No eligible monster templates found for theme. PreferredTags: %v, ExcludeTags: %v\n",
			options.PreferredTags, options.ExcludeTags)
	} else {
		fmt.Printf("Eligible monsters for spawning:\n")
		for _, template := range monsterTemplates {
			fmt.Printf("  - %s (weight: %d, level: %d)\n", template.ID, template.Weight, template.Level)
		}
	}

	// Place monsters throughout the dungeon
	monstersPlaced := 0
	maxAttempts := totalMonsters * 5 // Limit attempts to avoid infinite loop

	for attempts := 0; monstersPlaced < totalMonsters && attempts < maxAttempts; attempts++ {
		// Find an empty position for the monster
		x, y := p.findEmptyPosition(mapComp)

		// Skip if position is not valid (e.g., too close to player start)
		if !p.isValidMonsterPosition(mapComp, x, y) {
			continue
		}

		// Choose a monster template based on weighted distribution
		templateID := p.chooseMonsterTemplate(monsterTemplates)
		if templateID == "" {
			continue // No suitable templates found
		}

		// Create the monster
		_, err := p.entitySpawner.CreateEnemy(x, y, templateID)
		if err == nil {
			monstersPlaced++
		}
	}

	fmt.Printf("Placed %d monsters out of %d planned\n", monstersPlaced, totalMonsters)
}

// countRooms estimates the number of rooms in the dungeon
func (p *DungeonPopulator) countRooms(mapComp *components.MapComponent) int {
	// A simple heuristic: count floor tiles and divide by average room size
	floorTiles := 0
	averageRoomSize := 25 // Assumes average room is 5x5

	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				floorTiles++
			}
		}
	}

	roomEstimate := floorTiles / averageRoomSize
	if roomEstimate < 1 {
		return 1
	}
	return roomEstimate
}

// findEmptyPosition finds an unoccupied position for spawning
func (p *DungeonPopulator) findEmptyPosition(mapComp *components.MapComponent) (int, int) {
	for {
		x := p.rng.Intn(mapComp.Width)
		y := p.rng.Intn(mapComp.Height)

		if mapComp.Tiles[y][x] == components.TileFloor {
			return x, y
		}
	}
}

// isValidMonsterPosition checks if a position is valid for monster placement
func (p *DungeonPopulator) isValidMonsterPosition(mapComp *components.MapComponent, x, y int) bool {
	// Check if position is a floor tile
	if mapComp.Tiles[y][x] != components.TileFloor {
		return false
	}

	// Check if the position is occupied by another entity
	// (This would require checking entities with position components at x,y)
	// This is a simplified version - a real implementation would check for entities

	// Check if position is too close to player start (if we had that info)
	// For now, this is a placeholder

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
	// If we have preferred tags from JSON theme, use those as the primary filter
	if len(options.PreferredTags) > 0 {
		// Monster must match at least one of the preferred tags
		return p.hasPreferredTags(template, options.PreferredTags)
	}

	// Standard theme accepts all monsters
	if options.Theme == ThemeStandard {
		return true
	}

	// For legacy enum themes, check for specific theme matches
	switch options.Theme {
	case ThemeUndead:
		return p.hasTag(template, "undead")
	case ThemeGoblinoid:
		return p.hasTag(template, "goblinoid") || p.hasTag(template, "humanoid")
	case ThemeInsects:
		return p.hasTag(template, "insect") || p.hasTag(template, "vermin")
	case ThemeDemonic:
		return p.hasTag(template, "demon") || p.hasTag(template, "devil")
	case ThemeAbandoned:
		// Abandoned theme uses a mix of monsters but primarily vermin
		return p.hasTag(template, "vermin") || p.hasTag(template, "insect") ||
			p.hasTag(template, "undead")
	}

	// Default to allowing the monster if theme handling is not implemented
	return true
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
