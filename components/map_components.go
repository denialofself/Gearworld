package components

import (
	"image/color"
)

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

	// Box drawing wall tiles
	TileWallHorizontal  // ─
	TileWallVertical    // │
	TileWallTopLeft     // ┌
	TileWallTopRight    // ┐
	TileWallBottomLeft  // └
	TileWallBottomRight // ┘
	TileWallTeeLeft     // ├
	TileWallTeeRight    // ┤
	TileWallTeeTop      // ┬
	TileWallTeeBottom   // ┴
	TileWallCross       // ┼
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

	// Box drawing wall tile definitions (using light gray color)
	wallColor := color.RGBA{160, 160, 160, 255}
	mapping.Definitions[TileWallHorizontal] = NewTileDefinitionByPos(4, 12, wallColor)
	mapping.Definitions[TileWallVertical] = NewTileDefinitionByPos(3, 11, wallColor)
	mapping.Definitions[TileWallTopLeft] = NewTileDefinition('┌', wallColor)
	mapping.Definitions[TileWallTopRight] = NewTileDefinition('┐', wallColor)
	mapping.Definitions[TileWallBottomLeft] = NewTileDefinition('└', wallColor)
	mapping.Definitions[TileWallBottomRight] = NewTileDefinition('┘', wallColor)
	mapping.Definitions[TileWallTeeLeft] = NewTileDefinition('├', wallColor)
	mapping.Definitions[TileWallTeeRight] = NewTileDefinition('┤', wallColor)
	mapping.Definitions[TileWallTeeTop] = NewTileDefinition('┬', wallColor)
	mapping.Definitions[TileWallTeeBottom] = NewTileDefinition('┴', wallColor)
	mapping.Definitions[TileWallCross] = NewTileDefinition('┼', wallColor)

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
// This is implemented in generation/mapping_helper.go to avoid import cycles
func (m *MapComponent) IsWall(x, y int) bool {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return true // Out of bounds is considered a wall
	}
	// This will be replaced by a reference to the function in mapping_helper.go
	if IsWallFunc != nil {
		return IsWallFunc(m.Tiles[y][x])
	}
	return false
}

// IsWallFunc is a function pointer to hold the reference to the generation package's implementation
// This will be set by the generation package to avoid import cycles
var IsWallFunc func(tileType int) bool

// SetTile sets the tile at the given position
func (m *MapComponent) SetTile(x, y, tileType int) {
	if x >= 0 && x < m.Width && y >= 0 && y < m.Height {
		m.Tiles[y][x] = tileType
	}
}

// ApplyBoxDrawingWalls processes wall tiles and applies box drawing characters
// by delegating to the implementation in the generation package
func (m *MapComponent) ApplyBoxDrawingWalls() {
	// Import cycle prevention: we use a function value approach to call
	// the function from the generation package without directly importing it

	// This function will be set by the generation package during initialization
	// See generation/mapping_helper.go
	if ApplyBoxDrawingWallsFunc != nil {
		ApplyBoxDrawingWallsFunc(m)
	}
}

// ApplyBoxDrawingWallsFunc is a function pointer to hold the reference to the generation package's implementation
// This will be set by the generation package to avoid import cycles
var ApplyBoxDrawingWallsFunc func(*MapComponent)

// IsFloorType is a function pointer to hold the reference to the generation package's implementation
// This will be set by the generation package to avoid import cycles
var IsFloorTypeFunc func(tileType int) bool
