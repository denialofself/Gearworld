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
