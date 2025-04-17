package generation

import (
	"math/rand"
	"time"

	"ebiten-rogue/components"
)

// DungeonType enum to identify different dungeon generation methods
type DungeonType int

const (
	DungeonTypeRandom DungeonType = iota
	DungeonTypeSmallBSP
	DungeonTypeLargeBSP
	DungeonTypeSmallCellular
	DungeonTypeLargeCellular
)

// DungeonGenerator handles procedural generation of dungeon layouts
type DungeonGenerator struct {
	rng *rand.Rand
}

// NewDungeonGenerator creates a new dungeon generator
func NewDungeonGenerator() *DungeonGenerator {
	return &DungeonGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetSeed allows setting a specific seed for reproducible dungeons
func (g *DungeonGenerator) SetSeed(seed int64) {
	g.rng = rand.New(rand.NewSource(seed))
}

// GenerateRoomsAndCorridors creates random rooms and connects them with corridors
func (g *DungeonGenerator) GenerateRoomsAndCorridors(mapComp *components.MapComponent) {
	// Create a few random rooms
	numRooms := 5 + g.rng.Intn(5) // 5-9 rooms

	minRoomSize := 5
	maxRoomSize := 10

	var rooms [][4]int // Store rooms as [x, y, width, height]

	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	for i := 0; i < numRooms; i++ {
		// Random room size
		roomWidth := minRoomSize + g.rng.Intn(maxRoomSize-minRoomSize+1)
		roomHeight := minRoomSize + g.rng.Intn(maxRoomSize-minRoomSize+1)

		// Random room position (leaving space for walls)
		roomX := g.rng.Intn(mapComp.Width-roomWidth-1) + 1
		roomY := g.rng.Intn(mapComp.Height-roomHeight-1) + 1

		// Store the room
		rooms = append(rooms, [4]int{roomX, roomY, roomWidth, roomHeight})

		// Create the room
		for y := roomY; y < roomY+roomHeight; y++ {
			for x := roomX; x < roomX+roomWidth; x++ {
				mapComp.SetTile(x, y, components.TileFloor)
			}
		}

		// If this isn't the first room, connect it to the previous room
		if i > 0 {
			// Get the center of the current room
			currentX := roomX + roomWidth/2
			currentY := roomY + roomHeight/2

			// Get the center of the previous room
			prevRoom := rooms[i-1]
			prevX := prevRoom[0] + prevRoom[2]/2
			prevY := prevRoom[1] + prevRoom[3]/2

			// Create corridor between rooms
			g.CreateCorridor(mapComp, currentX, currentY, prevX, prevY)
		}
	}

	// Add features like water, lava, stairs, etc.
	g.AddFeatures(mapComp, rooms)
}

// FindEmptyPosition locates an unoccupied floor tile in the map
func (g *DungeonGenerator) FindEmptyPosition(mapComp *components.MapComponent) (int, int) {
	for {
		x := g.rng.Intn(mapComp.Width)
		y := g.rng.Intn(mapComp.Height)

		if !mapComp.IsWall(x, y) {
			return x, y
		}
	}
}

// FindFirstRoomInMap returns the coordinates of the first room found in the map
func (g *DungeonGenerator) FindFirstRoomInMap(mapComp *components.MapComponent) [][4]int {
	// Create a visited array to track which tiles we've checked
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	// Scan the map for floor tiles
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if !visited[y][x] && mapComp.Tiles[y][x] == components.TileFloor {
				// Found a floor tile, flood fill to find room boundaries
				minX, minY := x, y
				maxX, maxY := x, y
				g.floodFillRoom(mapComp, visited, x, y, &minX, &minY, &maxX, &maxY)
				return [][4]int{{minX, minY, maxX, maxY}}
			}
		}
	}
	return nil
}

// floodFillRoom performs a flood fill to find room boundaries
func (g *DungeonGenerator) floodFillRoom(mapComp *components.MapComponent, visited [][]bool, x, y int, minX, minY, maxX, maxY *int) {
	if x < 0 || x >= mapComp.Width || y < 0 || y >= mapComp.Height || visited[y][x] || mapComp.Tiles[y][x] != components.TileFloor {
		return
	}

	visited[y][x] = true
	if x < *minX {
		*minX = x
	}
	if x > *maxX {
		*maxX = x
	}
	if y < *minY {
		*minY = y
	}
	if y > *maxY {
		*maxY = y
	}

	// Recursively flood fill in all directions
	g.floodFillRoom(mapComp, visited, x+1, y, minX, minY, maxX, maxY)
	g.floodFillRoom(mapComp, visited, x-1, y, minX, minY, maxX, maxY)
	g.floodFillRoom(mapComp, visited, x, y+1, minX, minY, maxX, maxY)
	g.floodFillRoom(mapComp, visited, x, y-1, minX, minY, maxX, maxY)
}

// Generate creates a new dungeon layout in the provided map component
func (g *DungeonGenerator) Generate(mapComp *components.MapComponent, size DungeonSize) [][4]int {
	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	// Generate the layout based on size
	var rooms [][4]int
	switch size {
	case SizeSmall:
		rooms = g.generateSmallBSP(mapComp)
	case SizeLarge:
		rooms = g.generateLargeBSP(mapComp)
	case SizeHuge:
		rooms = g.generateHugeBSP(mapComp)
	default: // SizeNormal
		rooms = g.generateNormalBSP(mapComp)
	}

	// Add features like water, lava, stairs, etc.
	g.AddFeatures(mapComp, rooms)

	return rooms
}

// generateSmallBSP generates a small BSP dungeon
func (g *DungeonGenerator) generateSmallBSP(mapComp *components.MapComponent) [][4]int {
	// TODO: Implement BSP generation for small dungeons
	// For now, fall back to random rooms
	g.GenerateRoomsAndCorridors(mapComp)
	return g.FindFirstRoomInMap(mapComp)
}

// generateLargeBSP generates a large BSP dungeon
func (g *DungeonGenerator) generateLargeBSP(mapComp *components.MapComponent) [][4]int {
	// TODO: Implement BSP generation for large dungeons
	// For now, fall back to random rooms
	g.GenerateRoomsAndCorridors(mapComp)
	return g.FindFirstRoomInMap(mapComp)
}

// generateHugeBSP generates a huge BSP dungeon
func (g *DungeonGenerator) generateHugeBSP(mapComp *components.MapComponent) [][4]int {
	// TODO: Implement BSP generation for huge dungeons
	// For now, fall back to random rooms
	g.GenerateRoomsAndCorridors(mapComp)
	return g.FindFirstRoomInMap(mapComp)
}

// generateNormalBSP generates a normal-sized BSP dungeon
func (g *DungeonGenerator) generateNormalBSP(mapComp *components.MapComponent) [][4]int {
	// TODO: Implement BSP generation for normal dungeons
	// For now, fall back to random rooms
	g.GenerateRoomsAndCorridors(mapComp)
	return g.FindFirstRoomInMap(mapComp)
}

// findRooms finds all rooms in the generated dungeon
func (d *DungeonGenerator) findRooms(mapComp *components.MapComponent) [][4]int {
	var rooms [][4]int
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	// Find rooms by flood filling floor tiles
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if !visited[y][x] && mapComp.Tiles[y][x] == components.TileFloor {
				// Found a new room, flood fill to find its bounds
				minX, minY := x, y
				maxX, maxY := x, y
				d.floodFillRoom(mapComp, visited, x, y, &minX, &minY, &maxX, &maxY)
				rooms = append(rooms, [4]int{minX, minY, maxX, maxY})
			}
		}
	}
	return rooms
}
