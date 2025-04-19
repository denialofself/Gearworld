package components

import (
	"ebiten-rogue/ecs"
	"fmt"
	"image/color"
)

// TransitionData stores information about a transition tile
type TransitionData struct {
	TargetMapID     ecs.EntityID // ID of the map this transition leads to
	TargetX         int          // X position where player appears in target map
	TargetY         int          // Y position where player appears in target map
	IsBidirectional bool         // Whether the transition works both ways
}

// MapComponent stores the game map data
type MapComponent struct {
	Width       int
	Height      int
	Tiles       [][]int
	Visible     [][]bool                       // Track currently visible tiles
	Explored    [][]bool                       // Track tiles that have been seen at least once
	Transitions map[int]map[int]TransitionData // Maps (x,y) coordinates to transition data
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

	// Box drawing wall tiles - restart iota sequence at 9
	TileWallHorizontal  = iota + 1 // 9 ─
	TileWallVertical               // 10 │
	TileWallTopLeft                // 11 ┌
	TileWallTopRight               // 12 ┐
	TileWallBottomLeft             // 13 └
	TileWallBottomRight            // 14 ┘
	TileWallTeeLeft                // 15 ├
	TileWallTeeRight               // 16 ┤
	TileWallTeeTop                 // 17 ┬
	TileWallTeeBottom              // 18 ┴
	TileWallCross                  // 19 ┼

	// World map biome tiles - explicitly assign values to avoid issues
	TileWasteland     = 100
	TileDesert        = 101
	TileDarkForest    = 102
	TileMountains     = 103
	TileRuinedRailway = 104
	TileSubstation    = 105
	// New railway tile types for smooth turns
	TileRailwayHorizontal  = 106
	TileRailwayVertical    = 107
	TileRailwayTopLeft     = 108
	TileRailwayTopRight    = 109
	TileRailwayBottomLeft  = 110
	TileRailwayBottomRight = 111
	TileRailwayTeeLeft     = 112
	TileRailwayTeeRight    = 113
	TileRailwayTeeTop      = 114
	TileRailwayTeeBottom   = 115
	TileRailwayCross       = 116
	// Special entity tiles
	TileTrainSprite = 117 // Train sprite for player on world map
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
	mapping.Definitions[TileWallTopLeft] = NewTileDefinitionByPos(10, 13, wallColor)
	mapping.Definitions[TileWallTopRight] = NewTileDefinitionByPos(15, 11, wallColor)
	mapping.Definitions[TileWallBottomLeft] = NewTileDefinitionByPos(0, 12, wallColor)
	mapping.Definitions[TileWallBottomRight] = NewTileDefinitionByPos(9, 13, wallColor)
	mapping.Definitions[TileWallTeeLeft] = NewTileDefinitionByPos(3, 12, wallColor)
	mapping.Definitions[TileWallTeeRight] = NewTileDefinitionByPos(4, 11, wallColor)
	mapping.Definitions[TileWallTeeTop] = NewTileDefinitionByPos(2, 12, wallColor)
	mapping.Definitions[TileWallTeeBottom] = NewTileDefinitionByPos(1, 12, wallColor)
	mapping.Definitions[TileWallCross] = NewTileDefinitionByPos(5, 12, wallColor)

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
	// World map biome definitions - Using character-based references for testing
	mapping.Definitions[TileWasteland] = NewTileDefinitionByPos(1, 11, color.RGBA{150, 140, 100, 255})     // Wasteland: brownish gray
	mapping.Definitions[TileDesert] = NewTileDefinitionByPos(2, 11, color.RGBA{230, 210, 150, 255})        // Desert: light sand color
	mapping.Definitions[TileDarkForest] = NewTileDefinitionByPos(8, 1, color.RGBA{40, 80, 40, 255})        // Dark Forest: deep green
	mapping.Definitions[TileMountains] = NewTileDefinitionByPos(14, 1, color.RGBA{120, 120, 120, 255})     // Mountains: gray
	mapping.Definitions[TileRuinedRailway] = NewTileDefinitionByPos(13, 3, color.RGBA{100, 100, 110, 255}) // Railway: rusty tracks
	mapping.Definitions[TileSubstation] = NewTileDefinitionByPos(15, 0, color.RGBA{200, 200, 0, 255})      // Substation: industrial yellow
	// New railway tile definitions
	railwayColor := color.RGBA{184, 49, 42, 255}                                              // Same rusty color as ruined railway
	mapping.Definitions[TileRailwayHorizontal] = NewTileDefinitionByPos(4, 12, railwayColor)  // Box drawing horizontal
	mapping.Definitions[TileRailwayVertical] = NewTileDefinitionByPos(3, 11, railwayColor)    // Box drawing vertical
	mapping.Definitions[TileRailwayTopLeft] = NewTileDefinitionByPos(10, 13, railwayColor)    // Box drawing top left corner
	mapping.Definitions[TileRailwayTopRight] = NewTileDefinitionByPos(15, 11, railwayColor)   // Box drawing top right corner
	mapping.Definitions[TileRailwayBottomLeft] = NewTileDefinitionByPos(0, 12, railwayColor)  // Box drawing bottom left corner
	mapping.Definitions[TileRailwayBottomRight] = NewTileDefinitionByPos(9, 13, railwayColor) // Box drawing bottom right corner
	mapping.Definitions[TileRailwayTeeLeft] = NewTileDefinitionByPos(3, 12, railwayColor)     // Box drawing tee left
	mapping.Definitions[TileRailwayTeeRight] = NewTileDefinitionByPos(4, 11, railwayColor)    // Box drawing tee right
	mapping.Definitions[TileRailwayTeeTop] = NewTileDefinitionByPos(2, 12, railwayColor)      // Box drawing tee up
	mapping.Definitions[TileRailwayTeeBottom] = NewTileDefinitionByPos(1, 12, railwayColor)   // Box drawing tee down
	mapping.Definitions[TileRailwayCross] = NewTileDefinitionByPos(5, 12, railwayColor)       // Box drawing cross

	// Special entity tiles
	trainColor := color.RGBA{200, 200, 200, 255}                                     // Bright metallic color
	mapping.Definitions[TileTrainSprite] = NewTileDefinitionByPos(13, 3, trainColor) // Train sprite

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

// NewMapComponent creates a new map component with the specified dimensions
func NewMapComponent(width, height int) *MapComponent {
	m := &MapComponent{
		Width:       width,
		Height:      height,
		Tiles:       make([][]int, height),
		Visible:     make([][]bool, height),
		Explored:    make([][]bool, height),
		Transitions: make(map[int]map[int]TransitionData),
	}

	// Initialize the 2D arrays
	for y := 0; y < height; y++ {
		m.Tiles[y] = make([]int, width)
		m.Visible[y] = make([]bool, width)
		m.Explored[y] = make([]bool, width)
	}

	return m
}

// IsWall returns true if the tile at (x, y) is a wall
// Uses IsWallFunc from generation/mapping_helper.go
func (m *MapComponent) IsWall(x, y int) bool {
	// Out of bounds is always considered a wall
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return true
	}

	tileType := m.Tiles[y][x]

	// Use the function pointer from mapping_helper.go if available
	if IsWallFunc != nil {
		return IsWallFunc(tileType)
	}

	// Fallback to basic wall detection if IsWallFunc isn't set yet
	// This should only happen during early initialization
	return tileType == TileWall
}

// IsWallFunc is a function pointer set by generation/mapping_helper.go
// It implements full wall tile type detection to avoid import cycles
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

// DebugWallDetection tests wall detection on all tile types
// This can help verify that wall detection works properly during initialization
func DebugWallDetection() {
	fmt.Println("DEBUG: Testing wall detection on all tile types")

	// Test if a tile is correctly identified as a wall
	testWallType := func(name string, tileType int) {
		// First test the direct IsWallFunc if available
		var result bool
		var method string

		if IsWallFunc != nil {
			result = IsWallFunc(tileType)
			method = "IsWallFunc"
		} else {
			// Use our fallback logic
			result = tileType == TileWall
			method = "fallback"
		}

		fmt.Printf("%s (%d): %v using %s\n", name, tileType, result, method)
	}

	// Test basic types
	testWallType("TileWall", TileWall)
	testWallType("TileFloor", TileFloor)

	// Test wall variant types
	testWallType("TileWallHorizontal", TileWallHorizontal)
	testWallType("TileWallVertical", TileWallVertical)
	testWallType("TileWallTopLeft", TileWallTopLeft)
	testWallType("TileWallTopRight", TileWallTopRight)
	testWallType("TileWallBottomLeft", TileWallBottomLeft)
	testWallType("TileWallBottomRight", TileWallBottomRight)
	testWallType("TileWallTeeLeft", TileWallTeeLeft)
	testWallType("TileWallTeeRight", TileWallTeeRight)
	testWallType("TileWallTeeTop", TileWallTeeTop)
	testWallType("TileWallTeeBottom", TileWallTeeBottom)
	testWallType("TileWallCross", TileWallCross)
}

// AddTransition adds a transition at the specified coordinates
func (m *MapComponent) AddTransition(x, y int, targetMapID ecs.EntityID, targetX, targetY int, isBidirectional bool) {
	// Initialize the x map if it doesn't exist
	if _, exists := m.Transitions[x]; !exists {
		m.Transitions[x] = make(map[int]TransitionData)
	}

	// Add the transition data
	m.Transitions[x][y] = TransitionData{
		TargetMapID:     targetMapID,
		TargetX:         targetX,
		TargetY:         targetY,
		IsBidirectional: isBidirectional,
	}
}

// GetTransition returns the transition data at the specified coordinates
func (m *MapComponent) GetTransition(x, y int) (TransitionData, bool) {
	if xMap, exists := m.Transitions[x]; exists {
		if data, exists := xMap[y]; exists {
			return data, true
		}
	}
	return TransitionData{}, false
}

// RemoveTransition removes the transition at the specified coordinates
func (m *MapComponent) RemoveTransition(x, y int) {
	if xMap, exists := m.Transitions[x]; exists {
		delete(xMap, y)
		// Clean up empty x maps
		if len(xMap) == 0 {
			delete(m.Transitions, x)
		}
	}
}
