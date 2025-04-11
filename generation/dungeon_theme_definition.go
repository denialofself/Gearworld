package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DungeonThemeDefinition defines a complete theme for a dungeon
type DungeonThemeDefinition struct {
	ID          string   `json:"id"`           // Unique identifier for the theme
	Name        string   `json:"name"`         // Display name for the theme
	Description string   `json:"description"`  // Description of the theme
	Difficulty  int      `json:"difficulty"`   // Base difficulty level (1-10)
	Tags        []string `json:"tags"`         // Tags for monsters that fit this theme
	ExcludeTags []string `json:"exclude_tags"` // Tags for monsters that don't fit this theme

	// Visual theming
	WaterChance  float64 `json:"water_chance"` // Chance of water pools (0.0-1.0)
	LavaChance   float64 `json:"lava_chance"`  // Chance of lava pools (0.0-1.0)
	GrassChance  float64 `json:"grass_chance"` // Chance of grass patches (0.0-1.0)
	TreeChance   float64 `json:"tree_chance"`  // Chance of trees (0.0-1.0)
	SpecialTiles []struct {
		TileType string  `json:"tile_type"` // Type of special tile
		Chance   float64 `json:"chance"`    // Chance of this tile appearing (0.0-1.0)
	} `json:"special_tiles"` // Special tiles specific to this theme

	// Monster population
	DensityFactor         float64  `json:"density_factor"`           // Monster density (0.0-2.0, 1.0 = standard)
	HigherLevelChance     float64  `json:"higher_level_chance"`      // Chance for monsters from next level up (0.0-1.0)
	EvenHigherLevelChance float64  `json:"even_higher_level_chance"` // Chance for monsters two levels up (0.0-1.0)
	BossChance            float64  `json:"boss_chance"`              // Chance of a boss monster (0.0-1.0)
	BossTypes             []string `json:"boss_types"`               // Possible boss monster types
}

// DungeonThemeManager handles loading and managing dungeon themes from JSON files
type DungeonThemeManager struct {
	themes map[string]*DungeonThemeDefinition
}

// NewDungeonThemeManager creates a new theme manager
func NewDungeonThemeManager() *DungeonThemeManager {
	return &DungeonThemeManager{
		themes: make(map[string]*DungeonThemeDefinition),
	}
}

// LoadThemesFromDirectory loads all theme definition files from a directory
func (m *DungeonThemeManager) LoadThemesFromDirectory(directory string) error {
	files, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to read theme directory: %v", err)
	}

	for _, file := range files {
		if err := m.LoadThemeFromFile(file); err != nil {
			return fmt.Errorf("failed to load theme from %s: %v", filepath.Base(file), err)
		}
	}

	return nil
}

// LoadThemeFromFile loads a single theme definition from a JSON file
func (m *DungeonThemeManager) LoadThemeFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read theme file: %v", err)
	}

	var theme DungeonThemeDefinition
	if err := json.Unmarshal(data, &theme); err != nil {
		return fmt.Errorf("failed to parse theme JSON: %v", err)
	}

	// Validate the theme
	if theme.ID == "" {
		return fmt.Errorf("theme is missing ID")
	}

	// Add the theme to our collection
	m.themes[theme.ID] = &theme
	return nil
}

// GetTheme retrieves a theme by ID
func (m *DungeonThemeManager) GetTheme(id string) *DungeonThemeDefinition {
	return m.themes[id]
}

// GetAllThemes returns all loaded themes
func (m *DungeonThemeManager) GetAllThemes() []*DungeonThemeDefinition {
	result := make([]*DungeonThemeDefinition, 0, len(m.themes))
	for _, theme := range m.themes {
		result = append(result, theme)
	}
	return result
}

// GetThemesByDifficulty returns themes with the given difficulty level
func (m *DungeonThemeManager) GetThemesByDifficulty(difficulty int) []*DungeonThemeDefinition {
	var result []*DungeonThemeDefinition
	for _, theme := range m.themes {
		if theme.Difficulty == difficulty {
			result = append(result, theme)
		}
	}
	return result
}
