package generation

import (
	"ebiten-rogue/components"
)

// GenerateSmallCellularDungeon creates a standard size dungeon using cellular automata
func (g *DungeonGenerator) GenerateSmallCellularDungeon(mapComp *components.MapComponent) {
	g.generateCellularDungeon(mapComp, false)
}

// GenerateLargeCellularDungeon creates a large (10x10 screens) dungeon using cellular automata
func (g *DungeonGenerator) GenerateLargeCellularDungeon(mapComp *components.MapComponent) {
	g.generateCellularDungeon(mapComp, true)
}

// GenerateGiantCellularDungeon creates a 20x20 screen massive dungeon using cellular automata
func (g *DungeonGenerator) GenerateGiantCellularDungeon(mapComp *components.MapComponent) {
	// Initialize map with random walls (40% chance - slightly less than small/large for more open spaces)
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if g.rng.Float64() < 0.40 {
				mapComp.SetTile(x, y, components.TileWall)
			} else {
				mapComp.SetTile(x, y, components.TileFloor)
			}
		}
	}

	// Calculate section dimensions (20x20 grid)
	sectionWidth := mapComp.Width / 20
	sectionHeight := mapComp.Height / 20

	// Process each section separately
	for gridY := 0; gridY < 20; gridY++ {
		for gridX := 0; gridX < 20; gridX++ {
			startX := gridX * sectionWidth
			startY := gridY * sectionHeight

			// Run cellular automata on this section
			for iteration := 0; iteration < 3; iteration++ { // Fewer iterations for smaller sections
				// Create a copy of the current section
				newMap := make([][]int, sectionHeight)
				for y := range newMap {
					newMap[y] = make([]int, sectionWidth)
					for x := range newMap[y] {
						if startY+y < mapComp.Height && startX+x < mapComp.Width {
							newMap[y][x] = mapComp.Tiles[startY+y][startX+x]
						}
					}
				}

				// Apply cellular automata rules to section
				for y := 0; y < sectionHeight; y++ {
					for x := 0; x < sectionWidth; x++ {
						if startY+y >= mapComp.Height || startX+x >= mapComp.Width {
							continue
						}

						walls := g.countAdjacentWalls(mapComp, startX+x, startY+y)

						// Modified rules for giant dungeon to create more open spaces
						if walls > 5 {
							newMap[y][x] = components.TileWall
						} else if walls < 4 {
							newMap[y][x] = components.TileFloor
						}
					}
				}

				// Update the section in the main map
				for y := 0; y < sectionHeight; y++ {
					for x := 0; x < sectionWidth; x++ {
						if startY+y < mapComp.Height && startX+x < mapComp.Width {
							mapComp.Tiles[startY+y][startX+x] = newMap[y][x]
						}
					}
				}
			}

			// Clean up the section
			for y := 0; y < sectionHeight; y++ {
				for x := 0; x < sectionWidth; x++ {
					if startY+y >= mapComp.Height || startX+x >= mapComp.Width {
						continue
					}

					walls := g.countAdjacentWalls(mapComp, startX+x, startY+y)

					// Remove isolated walls and fill isolated floors
					if mapComp.Tiles[startY+y][startX+x] == components.TileWall && walls <= 2 {
						mapComp.SetTile(startX+x, startY+y, components.TileFloor)
					} else if mapComp.Tiles[startY+y][startX+x] == components.TileFloor && walls >= 7 {
						mapComp.SetTile(startX+x, startY+y, components.TileWall)
					}
				}
			}

			// Connect to neighboring sections if this isn't the first section
			if gridX > 0 || gridY > 0 {
				// Find a floor tile in current section
				var thisX, thisY int
				for attempts := 0; attempts < 100; attempts++ {
					testX := startX + g.rng.Intn(sectionWidth)
					testY := startY + g.rng.Intn(sectionHeight)

					if mapComp.Tiles[testY][testX] == components.TileFloor {
						thisX, thisY = testX, testY
						break
					}
				}

				// Create corridors to neighboring sections
				if gridX > 0 && gridY > 0 {
					// 40% chance to connect to both previous sections
					if g.rng.Intn(100) < 40 {
						// Connect to left section
						g.createHorizontalCorridor(mapComp, startX-1, thisX, thisY)
						// Connect to above section
						g.createVerticalCorridor(mapComp, startY-1, thisY, thisX)
					} else {
						// Choose one direction randomly
						if g.rng.Intn(2) == 0 {
							g.createHorizontalCorridor(mapComp, startX-1, thisX, thisY)
						} else {
							g.createVerticalCorridor(mapComp, startY-1, thisY, thisX)
						}
					}
				} else if gridX > 0 {
					g.createHorizontalCorridor(mapComp, startX-1, thisX, thisY)
				} else if gridY > 0 {
					g.createVerticalCorridor(mapComp, startY-1, thisY, thisX)
				}
			}
		}
	}

	// Find all significant open areas in the dungeon
	rooms := g.findAllOpenAreas(mapComp)

	// Add dungeon features
	g.AddFeatures(mapComp, rooms)
}

// generateCellularDungeon creates a dungeon using cellular automata rules
func (g *DungeonGenerator) generateCellularDungeon(mapComp *components.MapComponent, isLarge bool) {
	// Initialize map with random walls (45% chance)
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if g.rng.Float64() < 0.45 {
				mapComp.SetTile(x, y, components.TileWall)
			} else {
				mapComp.SetTile(x, y, components.TileFloor)
			}
		}
	}

	// Run cellular automata iterations
	iterations := 4
	if isLarge {
		iterations = 6 // More iterations for large maps
	}

	for i := 0; i < iterations; i++ {
		// Create a copy of the current state
		newMap := make([][]int, mapComp.Height)
		for y := range newMap {
			newMap[y] = make([]int, mapComp.Width)
			copy(newMap[y], mapComp.Tiles[y])
		}

		// Apply cellular automata rules
		for y := 0; y < mapComp.Height; y++ {
			for x := 0; x < mapComp.Width; x++ {
				walls := g.countAdjacentWalls(mapComp, x, y)

				// If a cell has more than 4 wall neighbors, it becomes a wall
				// If a cell has less than 4 wall neighbors, it becomes a floor
				if walls > 4 {
					newMap[y][x] = components.TileWall
				} else if walls < 4 {
					newMap[y][x] = components.TileFloor
				}
			}
		}

		// Update the map
		for y := 0; y < mapComp.Height; y++ {
			copy(mapComp.Tiles[y], newMap[y])
		}
	}

	// Clean up isolated walls and floors
	g.cleanupIsolatedTiles(mapComp)

	// Add features
	var rooms [][4]int
	if isLarge {
		// For large maps, divide into sections and find room-like open areas
		sectionWidth := mapComp.Width / 10
		sectionHeight := mapComp.Height / 10

		for gridY := 0; gridY < 10; gridY++ {
			for gridX := 0; gridX < 10; gridX++ {
				startX := gridX * sectionWidth
				startY := gridY * sectionHeight

				// Find the largest open area in this section
				room := g.findLargestOpenArea(mapComp, startX, startY, sectionWidth, sectionHeight)
				if room != nil {
					rooms = append(rooms, [4]int{room.X, room.Y, room.Width, room.Height})
				}
			}
		}
	} else {
		// For small maps, find all significant open areas
		rooms = g.findAllOpenAreas(mapComp)
	}

	// Add dungeon features to the identified rooms
	g.AddFeatures(mapComp, rooms)
}

// countAdjacentWalls counts the number of wall tiles around a position
func (g *DungeonGenerator) countAdjacentWalls(mapComp *components.MapComponent, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			newX, newY := x+dx, y+dy

			// Count edges as walls
			if newX < 0 || newX >= mapComp.Width || newY < 0 || newY >= mapComp.Height {
				count++
				continue
			}

			if mapComp.Tiles[newY][newX] == components.TileWall {
				count++
			}
		}
	}
	return count
}

// cleanupIsolatedTiles removes single isolated walls and floors
func (g *DungeonGenerator) cleanupIsolatedTiles(mapComp *components.MapComponent) {
	for y := 1; y < mapComp.Height-1; y++ {
		for x := 1; x < mapComp.Width-1; x++ {
			walls := g.countAdjacentWalls(mapComp, x, y)

			// Remove isolated walls (surrounded by floors)
			if mapComp.Tiles[y][x] == components.TileWall && walls <= 2 {
				mapComp.SetTile(x, y, components.TileFloor)
			}

			// Fill in isolated floors (surrounded by walls)
			if mapComp.Tiles[y][x] == components.TileFloor && walls >= 7 {
				mapComp.SetTile(x, y, components.TileWall)
			}
		}
	}
}

// findLargestOpenArea finds the largest contiguous floor area in a section
func (g *DungeonGenerator) findLargestOpenArea(mapComp *components.MapComponent, startX, startY, width, height int) *Room {
	visited := make([][]bool, height)
	for i := range visited {
		visited[i] = make([]bool, width)
	}

	var largestRoom *Room
	maxSize := 0

	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			if y >= mapComp.Height || x >= mapComp.Width {
				continue
			}

			if !visited[y-startY][x-startX] && mapComp.Tiles[y][x] == components.TileFloor {
				room := g.floodFill(mapComp, x, y, visited, startX, startY, width, height)
				size := room.Width * room.Height
				if size > maxSize {
					maxSize = size
					largestRoom = room
				}
			}
		}
	}

	return largestRoom
}

// findAllOpenAreas identifies all significant open areas in the map
func (g *DungeonGenerator) findAllOpenAreas(mapComp *components.MapComponent) [][4]int {
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	var rooms [][4]int
	minRoomSize := 16 // Minimum area to consider as a room

	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if !visited[y][x] && mapComp.Tiles[y][x] == components.TileFloor {
				room := g.floodFill(mapComp, x, y, visited, 0, 0, mapComp.Width, mapComp.Height)
				if room.Width*room.Height >= minRoomSize {
					rooms = append(rooms, [4]int{room.X, room.Y, room.Width, room.Height})
				}
			}
		}
	}

	return rooms
}

// floodFill performs a flood fill to find a contiguous area of floor tiles
func (g *DungeonGenerator) floodFill(mapComp *components.MapComponent, x, y int, visited [][]bool, startX, startY, width, height int) *Room {
	if x < startX || x >= startX+width || y < startY || y >= startY+height {
		return &Room{X: x, Y: y, Width: 0, Height: 0}
	}

	if visited[y-startY][x-startX] || mapComp.Tiles[y][x] != components.TileFloor {
		return &Room{X: x, Y: y, Width: 0, Height: 0}
	}

	visited[y-startY][x-startX] = true

	minX, maxX := x, x
	minY, maxY := y, y

	// Check adjacent tiles
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}

			newX, newY := x+dx, y+dy
			if newX >= startX && newX < startX+width && newY >= startY && newY < startY+height {
				room := g.floodFill(mapComp, newX, newY, visited, startX, startY, width, height)
				minX = min(minX, room.X)
				maxX = max(maxX, room.X+room.Width)
				minY = min(minY, room.Y)
				maxY = max(maxY, room.Y+room.Height)
			}
		}
	}

	return &Room{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}
