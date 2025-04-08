package components

import (
	"image/color"

	"ebiten-rogue/ecs"
)

// Define component IDs for our game
const (
	Position ecs.ComponentID = iota
	Renderable
	Player
	Stats
	Collision
	AI
	MapComponentID
	Appearance // New component for custom tile appearances
	Camera     // Camera component for viewport management
	// Chunk removed - no longer needed
)

// PositionComponent stores entity position
type PositionComponent struct {
	X, Y int
}

// RenderableComponent stores rendering information
type RenderableComponent struct {
	Char       rune        // The character in the tileset (for ASCII-based tiles)
	TileX      int         // X position in the tileset (for direct position access)
	TileY      int         // Y position in the tileset (for direct position access)
	UseTilePos bool        // Whether to use tile position instead of Char
	FG         color.Color // Foreground color
	BG         color.Color // Background color (optional)
}

// NewRenderableComponent creates a renderable component using a character code
func NewRenderableComponent(glyph rune, fg color.Color) *RenderableComponent {
	return &RenderableComponent{
		Char:       glyph,
		UseTilePos: false,
		FG:         fg,
		BG:         color.RGBA{0, 0, 0, 255}, // Default black background
	}
}

// NewRenderableComponentByPos creates a renderable component using direct position in the tileset
func NewRenderableComponentByPos(tileX, tileY int, fg color.Color) *RenderableComponent {
	return &RenderableComponent{
		TileX:      tileX,
		TileY:      tileY,
		UseTilePos: true,
		FG:         fg,
		BG:         color.RGBA{0, 0, 0, 255}, // Default black background
	}
}

// PlayerComponent indicates that an entity is controlled by the player
type PlayerComponent struct{}

// StatsComponent stores entity stats
type StatsComponent struct {
	Health          int
	MaxHealth       int
	Attack          int
	Defense         int
	Level           int
	Exp             int
	Recovery        int // Recovery points for action point regeneration
	ActionPoints    int // Current action points
	MaxActionPoints int // Maximum action points
}

// CollisionComponent indicates entity can collide with other entities
type CollisionComponent struct {
	Blocks bool // Whether this entity blocks movement
}

// AIComponent stores AI behavior information
type AIComponent struct {
	Type             string     // Type of AI: "random", "chase", "slow_chase", etc.
	SightRange       int        // How far the entity can see
	Target           uint64     // Target entity ID (usually the player)
	Path             []PathNode // Current path to target (if pathfinding)
	LastKnownTargetX int        // Last known X position of target
	LastKnownTargetY int        // Last known Y position of target
}

// PathNode represents a single point in a path
type PathNode struct {
	X, Y int
}

// MapComponent stores the game map data
type MapComponent struct {
	Width  int
	Height int
	Tiles  [][]int
}

// Tile types
const (
	TileFloor = iota
	TileWall
	TileDoor
	TileStairsDown
	TileStairsUp
	TileWater
	TileLava
	TileGrass
	TileTree
)

// TileDefinition describes the visual appearance of a tile type
type TileDefinition struct {
	Glyph      rune        // The character in the tileset (for ASCII-based tiles)
	TileX      int         // X position in the tileset (for direct position access)
	TileY      int         // Y position in the tileset (for direct position access)
	UseTilePos bool        // Whether to use tile position instead of Glyph
	FG         color.Color // Foreground color
	BG         color.Color // Background color (optional)
}

// NewTileDefinition creates a tile definition using a character code
func NewTileDefinition(glyph rune, fg color.Color) TileDefinition {
	return TileDefinition{
		Glyph:      glyph,
		UseTilePos: false,
		FG:         fg,
	}
}

// NewTileDefinitionByPos creates a tile definition using direct position in the tileset
func NewTileDefinitionByPos(tileX, tileY int, fg color.Color) TileDefinition {
	return TileDefinition{
		TileX:      tileX,
		TileY:      tileY,
		UseTilePos: true,
		FG:         fg,
	}
}

// TileMappingComponent maps tile types to their visual representation
type TileMappingComponent struct {
	Definitions map[int]TileDefinition
}

// NewTileMappingComponent creates a default tile mapping
func NewTileMappingComponent() *TileMappingComponent {
	mapping := &TileMappingComponent{
		Definitions: make(map[int]TileDefinition),
	}

	// Set up default tile definitions using character-based references
	mapping.Definitions[TileFloor] = NewTileDefinition('.', color.RGBA{64, 64, 64, 255})
	mapping.Definitions[TileWall] = NewTileDefinition('#', color.RGBA{128, 128, 128, 255})
	mapping.Definitions[TileDoor] = NewTileDefinition('+', color.RGBA{139, 69, 19, 255}) // Brown
	mapping.Definitions[TileStairsDown] = NewTileDefinition('>', color.RGBA{255, 255, 255, 255})
	mapping.Definitions[TileStairsUp] = NewTileDefinition('<', color.RGBA{255, 255, 255, 255})

	// Set up examples using position-based references
	// These reference specific tiles in the tileset by x,y coordinates

	// Example: Use the water waves symbol at position (4, 14) in the tileset for water
	mapping.Definitions[TileWater] = NewTileDefinitionByPos(7, 15, color.RGBA{0, 0, 255, 255}) // Blue

	// Example: Use the fire symbol at position (15, 10) for lava
	mapping.Definitions[TileLava] = NewTileDefinitionByPos(14, 7, color.RGBA{255, 0, 0, 255}) // Red

	// Example: Use a nice grass symbol at position (5, 3) for grass
	mapping.Definitions[TileGrass] = NewTileDefinitionByPos(0, 11, color.RGBA{0, 128, 0, 255}) // Green

	// Example: Use a tree symbol at position (6, 4) for trees
	mapping.Definitions[TileTree] = NewTileDefinitionByPos(8, 1, color.RGBA{0, 100, 0, 255}) // Dark green

	return mapping
}

// GetTileDefinition returns the visual definition for a given tile type
func (t *TileMappingComponent) GetTileDefinition(tileType int) TileDefinition {
	if def, exists := t.Definitions[tileType]; exists {
		return def
	}

	// Return a default if the tile type isn't defined
	return TileDefinition{
		Glyph: '?',
		FG:    color.RGBA{255, 0, 255, 255}, // Magenta for undefined tiles
	}
}

// NewMapComponent creates a new map with the given dimensions
func NewMapComponent(width, height int) *MapComponent {
	m := &MapComponent{
		Width:  width,
		Height: height,
		Tiles:  make([][]int, height),
	}

	// Initialize the tiles
	for y := 0; y < height; y++ {
		m.Tiles[y] = make([]int, width)
		for x := 0; x < width; x++ {
			// Start with walls everywhere
			m.Tiles[y][x] = TileWall
		}
	}

	return m
}

// IsWall returns true if the tile at (x, y) is a wall
func (m *MapComponent) IsWall(x, y int) bool {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return true // Out of bounds is considered a wall
	}
	return m.Tiles[y][x] == TileWall
}

// SetTile sets the tile at the given position
func (m *MapComponent) SetTile(x, y, tileType int) {
	if x >= 0 && x < m.Width && y >= 0 && y < m.Height {
		m.Tiles[y][x] = tileType
	}
}

// CameraComponent tracks the viewport position for map scrolling
type CameraComponent struct {
	X, Y   int    // Top-left position of the camera in the world
	Target uint64 // Entity ID that the camera follows (usually the player)
}

// NewCameraComponent creates a new camera component that follows the specified target
func NewCameraComponent(targetEntityID uint64) *CameraComponent {
	return &CameraComponent{
		X:      0,
		Y:      0,
		Target: targetEntityID,
	}
}
