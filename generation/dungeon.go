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

// FindFirstRoomInMap attempts to identify the first room in a dungeon map
// Returns a room as [x, y, width, height] or an empty slice if no room could be identified
func (g *DungeonGenerator) FindFirstRoomInMap(mapComp *components.MapComponent) []int {
	// Strategy: scan the top-left quarter of the map looking for a room
	// A room is defined as a contiguous area of floor tiles surrounded by walls
	searchWidth := mapComp.Width / 2
	searchHeight := mapComp.Height / 2

	// First, find a floor tile (potential room center)
	var startX, startY int
	foundStart := false

	for y := 5; y < searchHeight-5 && !foundStart; y++ {
		for x := 5; x < searchWidth-5 && !foundStart; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				// Check if this might be room center (surrounded by floor tiles)
				floorCount := 0
				for dy := -2; dy <= 2; dy++ {
					for dx := -2; dx <= 2; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < mapComp.Width && ny >= 0 && ny < mapComp.Height &&
							mapComp.Tiles[ny][nx] == components.TileFloor {
							floorCount++
						}
					}
				}

				// If we found enough floor tiles around, this is likely a room center
				if floorCount >= 20 { // At least 20 out of 25 tiles are floor
					startX, startY = x, y
					foundStart = true
					break
				}
			}
		}
	}

	if !foundStart {
		return []int{} // No suitable room found
	}

	// Now find the room boundaries by expanding from the center
	minX, minY := startX, startY
	maxX, maxY := startX, startY

	// Find left boundary
	for x := startX; x >= 0; x-- {
		if mapComp.IsWall(x, startY) {
			minX = x + 1
			break
		}
		if x == 0 {
			minX = 0
		}
	}

	// Find right boundary
	for x := startX; x < mapComp.Width; x++ {
		if mapComp.IsWall(x, startY) {
			maxX = x - 1
			break
		}
		if x == mapComp.Width-1 {
			maxX = mapComp.Width - 1
		}
	}

	// Find top boundary
	for y := startY; y >= 0; y-- {
		if mapComp.IsWall(startX, y) {
			minY = y + 1
			break
		}
		if y == 0 {
			minY = 0
		}
	}

	// Find bottom boundary
	for y := startY; y < mapComp.Height; y++ {
		if mapComp.IsWall(startX, y) {
			maxY = y - 1
			break
		}
		if y == mapComp.Height-1 {
			maxY = mapComp.Height - 1
		}
	}

	width := maxX - minX + 1
	height := maxY - minY + 1

	// Validate the room (must be reasonable size)
	if width < 4 || height < 4 || width > mapComp.Width/2 || height > mapComp.Height/2 {
		return []int{} // Not a valid room
	}

	return []int{minX, minY, width, height}
}
