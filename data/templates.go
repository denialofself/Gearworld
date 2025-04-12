package data

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"path/filepath"
)

// EntityTemplate represents a template for creating entities (monsters, NPCs, etc.)
type EntityTemplate struct {
	// Basic info
	ID          string `json:"id"`          // Unique identifier
	Name        string `json:"name"`        // Display name
	Description string `json:"description"` // Description text

	// Visual appearance
	TileX int    `json:"tileX"` // X position in the tileset
	TileY int    `json:"tileY"` // Y position in the tileset
	Color string `json:"color"` // Color in hex format (e.g. "#00FF00")

	// Stats
	Health          int `json:"health"`
	Attack          int `json:"attack"`
	Defense         int `json:"defense"`
	Level           int `json:"level"`
	XP              int `json:"xp"`              // XP awarded when killed
	Recovery        int `json:"recovery"`        // Recovery points for action point regeneration
	ActionPoints    int `json:"actionPoints"`    // Action points for the entity
	MaxActionPoints int `json:"maxActionPoints"` // Maximum action points
	HealingFactor   int `json:"healingFactor"`   // Healing factor for health regeneration

	// Behavior
	AIType      string   `json:"aiType"`      // Type of AI behavior
	Tags        []string `json:"tags"`        // Tags for categorization (e.g. "enemy", "npc", "boss")
	BlocksPath  bool     `json:"blocksPath"`  // Whether it blocks movement
	SpawnWeight int      `json:"spawnWeight"` // Relative chance of spawning (higher = more common)
}

// EntityTemplateManager manages all entity templates
type EntityTemplateManager struct {
	Templates     map[string]*EntityTemplate
	ItemTemplates map[string]*ItemTemplate
}

// NewEntityTemplateManager creates a new template manager
func NewEntityTemplateManager() *EntityTemplateManager {
	return &EntityTemplateManager{
		Templates:     make(map[string]*EntityTemplate),
		ItemTemplates: make(map[string]*ItemTemplate),
	}
}

// LoadTemplatesFromDirectory loads all JSON template files from a directory
func (m *EntityTemplateManager) LoadTemplatesFromDirectory(dirPath string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(dirPath, file.Name())
		if err := m.LoadTemplateFromFile(fullPath); err != nil {
			return fmt.Errorf("failed to load template from %s: %w", file.Name(), err)
		}
	}

	return nil
}

// LoadItemTemplatesFromDirectory loads all JSON item template files from a directory
func (m *EntityTemplateManager) LoadItemTemplatesFromDirectory(dirPath string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read item template directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(dirPath, file.Name())
		if err := m.LoadItemTemplateFromFile(fullPath); err != nil {
			return fmt.Errorf("failed to load item template from %s: %w", file.Name(), err)
		}
	}

	return nil
}

// LoadTemplateFromFile loads a single entity template from a JSON file
func (m *EntityTemplateManager) LoadTemplateFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var template EntityTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return err
	}

	// Validate required fields
	if template.ID == "" {
		return fmt.Errorf("template ID cannot be empty: %s", filePath)
	}

	// Add to templates map
	m.Templates[template.ID] = &template
	return nil
}

// LoadItemTemplateFromFile loads a single item template from a JSON file
func (m *EntityTemplateManager) LoadItemTemplateFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var template ItemTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return err
	}

	// Validate required fields
	if err := ValidateItemTemplate(&template); err != nil {
		return fmt.Errorf("invalid item template in %s: %w", filePath, err)
	}

	// Add to templates map
	m.ItemTemplates[template.ID] = &template
	return nil
}

// GetTemplate returns a template by ID
func (m *EntityTemplateManager) GetTemplate(id string) (*EntityTemplate, bool) {
	template, ok := m.Templates[id]
	return template, ok
}

// GetItemTemplate returns an item template by ID
func (m *EntityTemplateManager) GetItemTemplate(id string) (*ItemTemplate, bool) {
	template, ok := m.ItemTemplates[id]
	return template, ok
}

// ParseHexColor converts a hex string to a color.RGBA
func ParseHexColor(hex string) (c color.RGBA) {
	c.A = 0xff

	if len(hex) < 7 {
		return
	}

	format := "#%02x%02x%02x"
	_, err := fmt.Sscanf(hex, format, &c.R, &c.G, &c.B)
	if err != nil {
		return color.RGBA{255, 255, 255, 255} // Default white on error
	}

	return
}

// ItemTemplate defines a template for creating items
type ItemTemplate struct {
	ID          string   `json:"id"`          // Unique identifier for the item type
	Name        string   `json:"name"`        // Display name
	Description string   `json:"description"` // Item description
	ItemType    string   `json:"item_type"`   // Type of item: "weapon", "armor", "potion", etc.
	TileX       int      `json:"tile_x"`      // X position in the tileset
	TileY       int      `json:"tile_y"`      // Y position in the tileset
	Color       string   `json:"color"`       // Item color in hex format
	Value       int      `json:"value"`       // Base value/power of the item
	Weight      int      `json:"weight"`      // Weight of the item
	Tags        []string `json:"tags"`        // Additional tags for the item
}

// ValidateItemTemplate ensures that the item template has all required fields
func ValidateItemTemplate(template *ItemTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("item template missing ID")
	}
	if template.Name == "" {
		return fmt.Errorf("item template '%s' missing name", template.ID)
	}
	if template.ItemType == "" {
		return fmt.Errorf("item template '%s' missing item_type", template.ID)
	}
	return nil
}
