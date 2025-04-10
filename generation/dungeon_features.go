package generation

import (
	"ebiten-rogue/components"
)

// FeatureGenerator handles the addition of dungeon features
type FeatureGenerator struct {
	rng *DungeonGenerator
}

// AddFeatures adds all dungeon features to the map
func (g *DungeonGenerator) AddFeatures(mapComp *components.MapComponent, rooms [][4]int) {
	g.addPools(mapComp)
	g.addStairs(mapComp, rooms)
	g.addVegetation(mapComp)
}

// addPools adds water and lava pools to the dungeon
func (g *DungeonGenerator) addPools(mapComp *components.MapComponent) {
	// Add some water/lava pools in random locations (1-3 pools)
	pools := 1 + g.rng.Intn(3)
	for i := 0; i < pools; i++ {
		// Find an empty spot for the pool
		var poolX, poolY int
		for {
			poolX = g.rng.Intn(mapComp.Width-5) + 2
			poolY = g.rng.Intn(mapComp.Height-5) + 2
			if !mapComp.IsWall(poolX, poolY) {
				break
			}
		}

		// Determine if this is water or lava
		poolType := components.TileWater
		if g.rng.Intn(100) < 30 { // 30% chance for lava
			poolType = components.TileLava
		}

		// Create a small pool (3x3 to 5x5)
		poolSize := 3 + g.rng.Intn(3)
		for y := poolY; y < poolY+poolSize && y < mapComp.Height-1; y++ {
			for x := poolX; x < poolX+poolSize && x < mapComp.Width-1; x++ {
				if !mapComp.IsWall(x, y) && g.rng.Intn(100) < 70 { // Make pools irregular
					mapComp.SetTile(x, y, poolType)
				}
			}
		}
	}
}

// addStairs adds up and down staircases to the dungeon
func (g *DungeonGenerator) addStairs(mapComp *components.MapComponent, rooms [][4]int) {
	if len(rooms) == 0 {
		return
	}

	// Add stairs down to the next level in the last room
	lastRoom := rooms[len(rooms)-1]
	stairsX := lastRoom[0] + g.rng.Intn(lastRoom[2])
	stairsY := lastRoom[1] + g.rng.Intn(lastRoom[3])
	mapComp.SetTile(stairsX, stairsY, components.TileStairsDown)

	// Always add stairs up in the first room (player starting area)
	firstRoom := rooms[0]

	// Calculate the center of the first room for player spawn area
	centerX := firstRoom[0] + firstRoom[2]/2
	centerY := firstRoom[1] + firstRoom[3]/2

	// Place stairs up near the center (player starting position)
	// but not exactly at center to avoid blocking player spawn
	upX := centerX + (g.rng.Intn(3) - 1) // -1, 0, or +1 from center
	upY := centerY + (g.rng.Intn(3) - 1) // -1, 0, or +1 from center

	// Make sure stairs up don't overlap with stairs down
	if upX == stairsX && upY == stairsY {
		upX = centerX + 2 // Move it slightly away
	}

	// Ensure stairs are within room bounds
	upX = max(firstRoom[0]+1, min(upX, firstRoom[0]+firstRoom[2]-2))
	upY = max(firstRoom[1]+1, min(upY, firstRoom[1]+firstRoom[3]-2))

	mapComp.SetTile(upX, upY, components.TileStairsUp)
}

// addVegetation adds trees and other plant life to the dungeon
func (g *DungeonGenerator) addVegetation(mapComp *components.MapComponent) {
	// Add trees (about 1% of tiles)
	for i := 0; i < mapComp.Width*mapComp.Height/100; i++ {
		x := g.rng.Intn(mapComp.Width)
		y := g.rng.Intn(mapComp.Height)
		// Only place trees on floor tiles and not on stairs
		tileType := mapComp.Tiles[y][x]
		if tileType == components.TileFloor && g.rng.Intn(100) < 30 { // 30% chance on eligible tiles
			mapComp.SetTile(x, y, components.TileTree)
		}
	}
}

// CreateCorridor creates a corridor between two points
func (g *DungeonGenerator) CreateCorridor(mapComp *components.MapComponent, x1, y1, x2, y2 int) {
	// Randomly choose between horizontal-first or vertical-first
	if g.rng.Intn(2) == 0 {
		g.createHorizontalCorridor(mapComp, x1, x2, y1)
		g.createVerticalCorridor(mapComp, y1, y2, x2)
	} else {
		g.createVerticalCorridor(mapComp, y1, y2, x1)
		g.createHorizontalCorridor(mapComp, x1, x2, y2)
	}

	// Add a door at one end (20% chance)
	if g.rng.Intn(100) < 20 {
		doorX, doorY := x1, y1
		if g.rng.Intn(2) == 0 {
			doorX, doorY = x2, y2
		}

		// Place the door if it's within bounds and on a floor tile
		if doorX >= 0 && doorX < mapComp.Width && doorY >= 0 && doorY < mapComp.Height {
			if mapComp.Tiles[doorY][doorX] == components.TileFloor {
				mapComp.SetTile(doorX, doorY, components.TileDoor)
			}
		}
	}
}

// createHorizontalCorridor creates a horizontal corridor from x1 to x2 at y
func (g *DungeonGenerator) createHorizontalCorridor(mapComp *components.MapComponent, x1, x2, y int) {
	for x := min(x1, x2); x <= max(x1, x2); x++ {
		// Check map bounds
		if x >= 0 && x < mapComp.Width && y >= 0 && y < mapComp.Height {
			mapComp.SetTile(x, y, components.TileFloor)
		}
	}
}

// createVerticalCorridor creates a vertical corridor from y1 to y2 at x
func (g *DungeonGenerator) createVerticalCorridor(mapComp *components.MapComponent, y1, y2, x int) {
	for y := min(y1, y2); y <= max(y1, y2); y++ {
		// Check map bounds
		if x >= 0 && x < mapComp.Width && y >= 0 && y < mapComp.Height {
			mapComp.SetTile(x, y, components.TileFloor)
		}
	}
}
