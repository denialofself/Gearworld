package generation

import (
	"ebiten-rogue/components"
)

func init() {
	// Set the function pointers in components package to prevent import cycles
	components.ApplyBoxDrawingWallsFunc = ApplyBoxDrawingWalls
	components.IsWallFunc = IsAnyWallType
	components.IsFloorTypeFunc = IsFloorType
}

// Wall connection constants
const (
	WallConnectTop    = 1
	WallConnectRight  = 2
	WallConnectBottom = 4
	WallConnectLeft   = 8
)

// Box drawing wall tile lookup table
var WallTileLookup = map[int]int{
	0:  components.TileWall,            // No connections (isolated wall)
	1:  components.TileWallVertical,    // Top only
	2:  components.TileWallHorizontal,  // Right only
	3:  components.TileWallBottomLeft,  // Top and right
	4:  components.TileWallVertical,    // Bottom only
	5:  components.TileWallVertical,    // Top and bottom (vertical)
	6:  components.TileWallTopLeft,     // Right and bottom
	7:  components.TileWallTeeLeft,     // Top, right, bottom (missing left)
	8:  components.TileWallHorizontal,  // Left only
	9:  components.TileWallBottomRight, // Top and left
	10: components.TileWallHorizontal,  // Left and right (horizontal)
	11: components.TileWallTeeBottom,   // Top, left, right (missing bottom)
	12: components.TileWallTopRight,    // Left and bottom
	13: components.TileWallTeeRight,    // Top, left, bottom (missing right)
	14: components.TileWallTeeTop,      // Right, bottom, left (missing top)
	15: components.TileWallCross,       // All four neighbors
}

// ApplyBoxDrawingWalls processes the map and applies box drawing wall tiles
// to all walls that have at least one adjacent floor tile
func ApplyBoxDrawingWalls(mapComp *components.MapComponent) {
	// First pass: Identify perimeter walls (walls with at least one adjacent non-wall)
	perimeterWalls := make([][]bool, mapComp.Height)
	for y := 0; y < mapComp.Height; y++ {
		perimeterWalls[y] = make([]bool, mapComp.Width)
		for x := 0; x < mapComp.Width; x++ {
			if IsWallTile(mapComp.Tiles[y][x]) {
				// Check if it has any adjacent floor tiles
				if HasAdjacentFloor(mapComp, x, y) {
					perimeterWalls[y][x] = true
				}
			}
		}
	}

	// Second pass: Apply the appropriate wall tile to each perimeter wall
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if perimeterWalls[y][x] {
				maskValue := CalculateWallMask(mapComp, x, y)
				mapComp.Tiles[y][x] = WallTileLookup[maskValue]
			}
		}
	}
}

// HasAdjacentFloor checks if a position has at least one adjacent non-wall tile
func HasAdjacentFloor(mapComp *components.MapComponent, x, y int) bool {
	// Check all 4 cardinal directions for floor tiles
	if y > 0 && !IsWallOrDoor(mapComp, x, y-1) { // Top
		return true
	}
	if x < mapComp.Width-1 && !IsWallOrDoor(mapComp, x+1, y) { // Right
		return true
	}
	if y < mapComp.Height-1 && !IsWallOrDoor(mapComp, x, y+1) { // Bottom
		return true
	}
	if x > 0 && !IsWallOrDoor(mapComp, x-1, y) { // Left
		return true
	}
	return false
}

// CalculateWallMask calculates the bitmask value for a wall tile
// based on which adjacent tiles are walls
func CalculateWallMask(mapComp *components.MapComponent, x, y int) int {
	mask := 0

	// Check for walls in each direction and set appropriate bits
	if y > 0 && IsWallOrDoor(mapComp, x, y-1) { // Top
		mask |= WallConnectTop
	}
	if x < mapComp.Width-1 && IsWallOrDoor(mapComp, x+1, y) { // Right
		mask |= WallConnectRight
	}
	if y < mapComp.Height-1 && IsWallOrDoor(mapComp, x, y+1) { // Bottom
		mask |= WallConnectBottom
	}
	if x > 0 && IsWallOrDoor(mapComp, x-1, y) { // Left
		mask |= WallConnectLeft
	}

	return mask
}

// IsWallTile checks if a tile is a basic wall tile (not already a special wall type)
func IsWallTile(tileType int) bool {
	return tileType == components.TileWall
}

// IsWallOrDoor checks if a tile is any type of wall or a door
func IsWallOrDoor(mapComp *components.MapComponent, x, y int) bool {
	// Bounds check
	if x < 0 || x >= mapComp.Width || y < 0 || y >= mapComp.Height {
		return true // Consider out-of-bounds as walls
	}

	// Check if it's a wall or door
	tileType := mapComp.Tiles[y][x]
	return IsAnyWallType(tileType) || tileType == components.TileDoor
}

// IsAnyWallType checks if a tile is any type of wall
func IsAnyWallType(tileType int) bool {
	return tileType == components.TileWall ||
		tileType == components.TileWallHorizontal ||
		tileType == components.TileWallVertical ||
		tileType == components.TileWallTopLeft ||
		tileType == components.TileWallTopRight ||
		tileType == components.TileWallBottomLeft ||
		tileType == components.TileWallBottomRight ||
		tileType == components.TileWallTeeLeft ||
		tileType == components.TileWallTeeRight ||
		tileType == components.TileWallTeeTop ||
		tileType == components.TileWallTeeBottom ||
		tileType == components.TileWallCross
}

// IsFloorType checks if a tile is a floor-type (not wall or door)
func IsFloorType(tileType int) bool {
	return tileType != components.TileWall &&
		tileType != components.TileDoor &&
		tileType != components.TileWallHorizontal &&
		tileType != components.TileWallVertical &&
		tileType != components.TileWallTopLeft &&
		tileType != components.TileWallTopRight &&
		tileType != components.TileWallBottomLeft &&
		tileType != components.TileWallBottomRight &&
		tileType != components.TileWallTeeLeft &&
		tileType != components.TileWallTeeRight &&
		tileType != components.TileWallTeeTop &&
		tileType != components.TileWallTeeBottom &&
		tileType != components.TileWallCross
}
