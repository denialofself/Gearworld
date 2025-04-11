package generation

import (
	"ebiten-rogue/components"
	"fmt"
)

// init sets up function references to avoid import cycles between packages
// This is called automatically when the package is first used
func init() {
	// Set the function pointers in the components package
	components.ApplyBoxDrawingWallsFunc = ApplyBoxDrawingWalls
	components.IsWallFunc = IsAnyWallType
	components.IsFloorTypeFunc = IsFloorType

	// Log that we've initialized the mapping helper
	fmt.Println("INFO: mapping_helper.go initialized - Wall detection functions are now available")
}

// Wall connection constants used for box drawing walls
const (
	WallConnectTop    = 1
	WallConnectRight  = 2
	WallConnectBottom = 4
	WallConnectLeft   = 8
)

// Box drawing wall tile lookup table
var WallTileLookup = map[int]int{
	0:  components.TileWall,            // No connections (isolated wall)
	1:  components.TileWallVertical,    // Top only (11: │)
	2:  components.TileWallHorizontal,  // Right only (10: ─)
	3:  components.TileWallBottomLeft,  // Top and right (13: └)
	4:  components.TileWallVertical,    // Bottom only (11: │)
	5:  components.TileWallVertical,    // Top and bottom (vertical) (11: │)
	6:  components.TileWallTopLeft,     // Right and bottom (11: ┌)
	7:  components.TileWallTeeLeft,     // Top, right, bottom (missing left) (15: ├)
	8:  components.TileWallHorizontal,  // Left only (10: ─)
	9:  components.TileWallBottomRight, // Top and left (14: ┘)
	10: components.TileWallHorizontal,  // Left and right (horizontal) (10: ─)
	11: components.TileWallTeeBottom,   // Top, left, right (missing bottom) (18: ┴)
	12: components.TileWallTopRight,    // Left and bottom (12: ┐)
	13: components.TileWallTeeRight,    // Top, left, bottom (missing right) (16: ┤)
	14: components.TileWallTeeTop,      // Right, bottom, left (missing top) (17: ┬)
	15: components.TileWallCross,       // All four neighbors (19: ┼)
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
// This is exported to components.IsWallFunc to avoid import cycles
func IsAnyWallType(tileType int) bool {
	return tileType == components.TileWall ||
		tileType == components.TileWallHorizontal || // 9 ─
		tileType == components.TileWallVertical || // 10 │
		tileType == components.TileWallTopLeft || // 11 ┌
		tileType == components.TileWallTopRight || // 12 ┐
		tileType == components.TileWallBottomLeft || // 13 └
		tileType == components.TileWallBottomRight || // 14 ┘
		tileType == components.TileWallTeeLeft || // 15 ├
		tileType == components.TileWallTeeRight || // 16 ┤
		tileType == components.TileWallTeeTop || // 17 ┬
		tileType == components.TileWallTeeBottom || // 18 ┴
		tileType == components.TileWallCross // 19 ┼
}

// IsFloorType checks if a tile is a floor-type (not wall or door)
// This is exported to components.IsFloorTypeFunc to avoid import cycles
func IsFloorType(tileType int) bool {
	return tileType != components.TileWall &&
		tileType != components.TileDoor &&
		tileType != components.TileWallHorizontal && // 9 ─
		tileType != components.TileWallVertical && // 10 │
		tileType != components.TileWallTopLeft && // 11 ┌
		tileType != components.TileWallTopRight && // 12 ┐
		tileType != components.TileWallBottomLeft && // 13 └
		tileType != components.TileWallBottomRight && // 14 ┘
		tileType != components.TileWallTeeLeft && // 15 ├
		tileType != components.TileWallTeeRight && // 16 ┤
		tileType != components.TileWallTeeTop && // 17 ┬
		tileType != components.TileWallTeeBottom && // 18 ┴
		tileType != components.TileWallCross // 19 ┼
}
