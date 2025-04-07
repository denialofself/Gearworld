// filepath: d:\Temp\ebiten-rogue\generation\dungeon.go
package generation

import (
	"math/rand"
	"time"

	"ebiten-rogue/components"
)

// DungeonType enum to identify different dungeon generation methods
type DungeonType int

const (
	DungeonTypeRandom   DungeonType = iota // Original random rooms
	DungeonTypeSmallBSP                    // Small BSP-based dungeon
	DungeonTypeLargeBSP                    // Large BSP-based dungeon (10x10 screens)
)

// BSPNode represents a node in the binary space partitioning tree
type BSPNode struct {
	X, Y, Width, Height int
	Left, Right         *BSPNode
	Room                *Room
	Parent              *BSPNode
	corridorStart       []int
	corridorEnd         []int
}

// Room represents a room within the dungeon
type Room struct {
	X, Y, Width, Height int
}

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

		// Create the room with floor tiles
		for y := roomY; y < roomY+roomHeight; y++ {
			for x := roomX; x < roomX+roomWidth; x++ {
				// Use floor tiles for the room interior
				mapComp.SetTile(x, y, components.TileFloor)

				// Add some random grass tiles (5% chance)
				if g.rng.Intn(100) < 5 {
					mapComp.SetTile(x, y, components.TileGrass)
				}
			}
		}

		// If this isn't the first room, connect it to the previous room
		if i > 0 {
			// Connect rooms with corridors
			// Get the center of the current room
			currentX := roomX + roomWidth/2
			currentY := roomY + roomHeight/2

			// Get the center of the previous room
			prevRoom := rooms[i-1]
			prevX := prevRoom[0] + prevRoom[2]/2
			prevY := prevRoom[1] + prevRoom[3]/2

			// Create corridors (horizontal then vertical or vice versa)
			if g.rng.Intn(2) == 0 {
				// Horizontal then vertical
				g.createHorizontalCorridor(mapComp, currentX, prevX, currentY)
				g.createVerticalCorridor(mapComp, currentY, prevY, prevX)
			} else {
				// Vertical then horizontal
				g.createVerticalCorridor(mapComp, currentY, prevY, currentX)
				g.createHorizontalCorridor(mapComp, currentX, prevX, prevY)
			}

			// Add a door at one end of the corridor (20% chance)
			if g.rng.Intn(100) < 20 {
				// Choose a position for the door
				doorX, doorY := prevX, prevY
				if g.rng.Intn(2) == 0 {
					// Place door at the current room entrance
					doorX, doorY = currentX, currentY
				}

				// Place the door
				mapComp.SetTile(doorX, doorY, components.TileDoor)
			}
		}
	}

	g.addDungeonFeatures(mapComp, rooms)
}

// addDungeonFeatures adds decorative and functional elements to the dungeon
func (g *DungeonGenerator) addDungeonFeatures(mapComp *components.MapComponent, rooms [][4]int) {
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

	// Add stairs down to the next level
	var stairsX, stairsY int
	// Place the stairs in the last room
	if len(rooms) > 0 {
		lastRoom := rooms[len(rooms)-1]
		stairsX = lastRoom[0] + g.rng.Intn(lastRoom[2])
		stairsY = lastRoom[1] + g.rng.Intn(lastRoom[3])
		mapComp.SetTile(stairsX, stairsY, components.TileStairsDown)
	}

	// Add stairs up (if not the first level)
	if g.rng.Intn(100) < 50 { // 50% chance to add stairs up
		// Place the stairs up in the first room
		if len(rooms) > 0 {
			firstRoom := rooms[0]
			upX := firstRoom[0] + g.rng.Intn(firstRoom[2])
			upY := firstRoom[1] + g.rng.Intn(firstRoom[3])
			// Don't place on top of the down stairs
			if upX != stairsX || upY != stairsY {
				mapComp.SetTile(upX, upY, components.TileStairsUp)
			}
		}
	}

	// Add some trees around the map
	for i := 0; i < mapComp.Width*mapComp.Height/100; i++ { // About 1% of tiles become trees
		x := g.rng.Intn(mapComp.Width)
		y := g.rng.Intn(mapComp.Height)
		// Only place trees on floor tiles and not on stairs
		tileType := mapComp.Tiles[y][x]
		if tileType == components.TileFloor && g.rng.Intn(100) < 30 { // 30% chance on eligible tiles
			mapComp.SetTile(x, y, components.TileTree)
		}
	}
}

// GenerateBSPDungeon creates a dungeon using binary space partitioning
func (g *DungeonGenerator) GenerateBSPDungeon(mapComp *components.MapComponent) {
	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	// Create the root node for the entire map
	root := &BSPNode{
		X:      0,
		Y:      0,
		Width:  mapComp.Width,
		Height: mapComp.Height,
	}

	// Recursively split the space
	g.splitNode(root, 0, 6) // Maximum depth of 6

	// Generate rooms within the leaf nodes
	g.createRoomsInLeaves(root)

	// Connect rooms together
	g.connectRooms(root)

	// Draw the rooms and corridors on the map
	g.drawBSPDungeon(root, mapComp)

	// Add features (water, lava, stairs, trees, etc.)
	var rooms [][4]int
	g.collectRooms(root, &rooms)
	g.addDungeonFeatures(mapComp, rooms)
}

// splitNode recursively splits a BSP node into two child nodes
func (g *DungeonGenerator) splitNode(node *BSPNode, depth, maxDepth int) bool {
	// Stop recursion if we've reached maximum depth
	if depth >= maxDepth {
		return false
	}

	// If node already has children, process them
	if node.Left != nil || node.Right != nil {
		if node.Left != nil {
			g.splitNode(node.Left, depth+1, maxDepth)
		}
		if node.Right != nil {
			g.splitNode(node.Right, depth+1, maxDepth)
		}
		return true
	}

	// Decide whether to split horizontally or vertically
	// If width > 25% larger than height, split vertically
	// If height > 25% larger than width, split horizontally
	// Otherwise, choose randomly
	horizontal := false
	if float64(node.Width) > float64(node.Height)*1.25 {
		horizontal = false // Split vertically
	} else if float64(node.Height) > float64(node.Width)*1.25 {
		horizontal = true // Split horizontally
	} else {
		horizontal = g.rng.Intn(2) == 0 // Random choice
	}

	// Calculate minimum split dimension
	minSize := 10 // Minimum size of a node after splitting

	// Check if the node is too small to split
	if (horizontal && node.Height < 2*minSize+1) || (!horizontal && node.Width < 2*minSize+1) {
		return false
	}

	// Calculate split position (leaving at least minSize on each side)
	var splitPos int
	if horizontal {
		// Ensure we have at least 1 position to randomize
		splitRange := node.Height - 2*minSize
		if splitRange <= 0 {
			splitRange = 1
		}
		splitPos = minSize + g.rng.Intn(splitRange)

		// Create child nodes
		node.Left = &BSPNode{
			X:      node.X,
			Y:      node.Y,
			Width:  node.Width,
			Height: splitPos,
			Parent: node,
		}
		node.Right = &BSPNode{
			X:      node.X,
			Y:      node.Y + splitPos,
			Width:  node.Width,
			Height: node.Height - splitPos,
			Parent: node,
		}
	} else {
		// Ensure we have at least 1 position to randomize
		splitRange := node.Width - 2*minSize
		if splitRange <= 0 {
			splitRange = 1
		}
		splitPos = minSize + g.rng.Intn(splitRange)

		// Create child nodes
		node.Left = &BSPNode{
			X:      node.X,
			Y:      node.Y,
			Width:  splitPos,
			Height: node.Height,
			Parent: node,
		}
		node.Right = &BSPNode{
			X:      node.X + splitPos,
			Y:      node.Y,
			Width:  node.Width - splitPos,
			Height: node.Height,
			Parent: node,
		}
	}

	// Continue splitting the children
	g.splitNode(node.Left, depth+1, maxDepth)
	g.splitNode(node.Right, depth+1, maxDepth)

	return true
}

// createRoomsInLeaves generates rooms in the leaf nodes of the BSP tree
func (g *DungeonGenerator) createRoomsInLeaves(node *BSPNode) {
	// If this is a leaf node (no children)
	if node.Left == nil && node.Right == nil {
		// Room dimensions (leaving space for walls)
		minPadding := 1
		maxPadding := 3

		// Random padding on each side
		paddingLeft := minPadding + g.rng.Intn(maxPadding)
		paddingTop := minPadding + g.rng.Intn(maxPadding)

		// Calculate remaining space
		remainingWidth := node.Width - 2*paddingLeft
		remainingHeight := node.Height - 2*paddingTop

		// Ensure minimum room size
		if remainingWidth < 4 || remainingHeight < 4 {
			return // Not enough space for a room
		}

		// Create the room
		roomWidth := max(4, remainingWidth)
		roomHeight := max(4, remainingHeight)

		roomX := node.X + paddingLeft
		roomY := node.Y + paddingTop

		// Store the room in the node
		node.Room = &Room{
			X:      roomX,
			Y:      roomY,
			Width:  roomWidth,
			Height: roomHeight,
		}

		return
	}

	// Process children if this is not a leaf
	if node.Left != nil {
		g.createRoomsInLeaves(node.Left)
	}
	if node.Right != nil {
		g.createRoomsInLeaves(node.Right)
	}
}

// connectRooms creates corridors between adjacent rooms in the BSP tree
func (g *DungeonGenerator) connectRooms(node *BSPNode) {
	// If this is an internal node with both children
	if node.Left != nil && node.Right != nil {
		// Find rooms to connect in each subtree
		leftRoom := g.findRoom(node.Left)
		rightRoom := g.findRoom(node.Right)

		// If both subtrees have rooms, connect them
		if leftRoom != nil && rightRoom != nil {
			// Calculate centers of rooms
			leftCenterX := leftRoom.X + leftRoom.Width/2
			leftCenterY := leftRoom.Y + leftRoom.Height/2

			rightCenterX := rightRoom.X + rightRoom.Width/2
			rightCenterY := rightRoom.Y + rightRoom.Height/2

			// Store corridor info in the parent node (for later drawing)
			node.corridorStart = []int{leftCenterX, leftCenterY}
			node.corridorEnd = []int{rightCenterX, rightCenterY}
		}

		// Continue connecting rooms in subtrees
		g.connectRooms(node.Left)
		g.connectRooms(node.Right)
	}
}

// CorridorStart and CorridorEnd for the BSPNode
func (node *BSPNode) CorridorStart() []int {
	// Implementation to retrieve corridor start point
	return node.corridorStart
}

func (node *BSPNode) CorridorEnd() []int {
	// Implementation to retrieve corridor end point
	return node.corridorEnd
}

// findRoom finds a room in the subtree rooted at the given node
func (g *DungeonGenerator) findRoom(node *BSPNode) *Room {
	// If this node has a room, return it
	if node.Room != nil {
		return node.Room
	}

	// Otherwise, recursively search in children
	var room *Room
	if node.Left != nil {
		room = g.findRoom(node.Left)
		if room != nil {
			return room
		}
	}

	if node.Right != nil {
		room = g.findRoom(node.Right)
		if room != nil {
			return room
		}
	}

	// No room found in this subtree
	return nil
}

// drawBSPDungeon draws rooms and corridors from the BSP tree onto the map
func (g *DungeonGenerator) drawBSPDungeon(node *BSPNode, mapComp *components.MapComponent) {
	// Draw the room if there is one
	if node.Room != nil {
		for y := node.Room.Y; y < node.Room.Y+node.Room.Height; y++ {
			for x := node.Room.X; x < node.Room.X+node.Room.Width; x++ {
				// Check bounds to prevent array out of bounds
				if x >= 0 && x < mapComp.Width && y >= 0 && y < mapComp.Height {
					mapComp.SetTile(x, y, components.TileFloor)

					// Add grass tiles occasionally (5% chance)
					if g.rng.Intn(100) < 5 {
						mapComp.SetTile(x, y, components.TileGrass)
					}
				}
			}
		}
	}

	// Draw corridor connecting this node's children
	if node.CorridorStart() != nil && node.CorridorEnd() != nil {
		x1, y1 := node.CorridorStart()[0], node.CorridorStart()[1]
		x2, y2 := node.CorridorEnd()[0], node.CorridorEnd()[1]

		// Create corridors (either horizontal-then-vertical or vertical-then-horizontal)
		if g.rng.Intn(2) == 0 {
			// Horizontal then vertical
			g.createHorizontalCorridor(mapComp, x1, x2, y1)
			g.createVerticalCorridor(mapComp, y1, y2, x2)
		} else {
			// Vertical then horizontal
			g.createVerticalCorridor(mapComp, y1, y2, x1)
			g.createHorizontalCorridor(mapComp, x1, x2, y2)
		}

		// Add a door at one end of the corridor (20% chance)
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

	// Recursively draw children
	if node.Left != nil {
		g.drawBSPDungeon(node.Left, mapComp)
	}
	if node.Right != nil {
		g.drawBSPDungeon(node.Right, mapComp)
	}
}

// collectRooms gathers all rooms from the BSP tree for later use
func (g *DungeonGenerator) collectRooms(node *BSPNode, rooms *[][4]int) {
	if node.Room != nil {
		*rooms = append(*rooms, [4]int{
			node.Room.X,
			node.Room.Y,
			node.Room.Width,
			node.Room.Height,
		})
	}

	if node.Left != nil {
		g.collectRooms(node.Left, rooms)
	}
	if node.Right != nil {
		g.collectRooms(node.Right, rooms)
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

// GenerateLargeBSPDungeon creates a 10x10 screen large dungeon using BSP
func (g *DungeonGenerator) GenerateLargeBSPDungeon(mapComp *components.MapComponent) {
	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	// Calculate section dimensions
	sectionWidth := mapComp.Width / 10
	sectionHeight := mapComp.Height / 10

	// Generate each section
	for gridY := 0; gridY < 10; gridY++ {
		for gridX := 0; gridX < 10; gridX++ {
			// Calculate section boundaries
			startX := gridX * sectionWidth
			startY := gridY * sectionHeight

			// Create a BSP node for this section
			sectionNode := &BSPNode{
				X:      startX,
				Y:      startY,
				Width:  sectionWidth,
				Height: sectionHeight,
			}

			// Split this section
			g.splitNode(sectionNode, 0, 3) // Lower max depth for smaller sections

			// Create rooms in this section
			g.createRoomsInLeaves(sectionNode)

			// Connect rooms in this section
			g.connectRooms(sectionNode)

			// Draw this section
			g.drawBSPDungeon(sectionNode, mapComp)

			// If not the first section, connect to a neighboring section
			if gridX > 0 || gridY > 0 {
				// Find a floor tile in this section
				var thisX, thisY int
				for attempts := 0; attempts < 100; attempts++ {
					testX := startX + g.rng.Intn(sectionWidth)
					testY := startY + g.rng.Intn(sectionHeight)

					if mapComp.Tiles[testY][testX] == components.TileFloor {
						thisX, thisY = testX, testY
						break
					}
				}

				// Find a neighbor section to connect to
				var neighborX, neighborY int

				// Choose either left or above neighbor (if available)
				if gridX > 0 && gridY > 0 {
					// Choose randomly between left or above neighbor
					if g.rng.Intn(2) == 0 {
						neighborX = startX - 1 // Connect to left section
						neighborY = thisY
					} else {
						neighborX = thisX
						neighborY = startY - 1 // Connect to above section
					}
				} else if gridX > 0 {
					// Connect to left section
					neighborX = startX - 1
					neighborY = thisY
				} else if gridY > 0 {
					// Connect to above section
					neighborX = thisX
					neighborY = startY - 1
				}

				// Create corridor to neighboring section
				if gridX > 0 || gridY > 0 {
					// Find a floor tile in the neighboring section
					var foundNeighborTile bool
					for attempts := 0; attempts < 100; attempts++ {
						// Search in a spiral pattern from neighborX, neighborY
						for radius := 1; radius < 10; radius++ {
							for offY := -radius; offY <= radius; offY++ {
								for offX := -radius; offX <= radius; offX++ {
									testX := neighborX + offX
									testY := neighborY + offY

									// Check bounds
									if testX >= 0 && testX < mapComp.Width &&
										testY >= 0 && testY < mapComp.Height {
										// If we found a floor tile in the neighboring section
										if mapComp.Tiles[testY][testX] == components.TileFloor {
											// Create corridor
											if g.rng.Intn(2) == 0 {
												// Horizontal then vertical
												g.createHorizontalCorridor(mapComp, thisX, testX, thisY)
												g.createVerticalCorridor(mapComp, thisY, testY, testX)
											} else {
												// Vertical then horizontal
												g.createVerticalCorridor(mapComp, thisY, testY, thisX)
												g.createHorizontalCorridor(mapComp, thisX, testX, testY)
											}

											foundNeighborTile = true
											break
										}
									}
								}
								if foundNeighborTile {
									break
								}
							}
							if foundNeighborTile {
								break
							}
						}
						if foundNeighborTile {
							break
						}
					}
				}
			}
		}
	}

	// Add features to the dungeon
	var allRooms [][4]int
	g.addDungeonFeatures(mapComp, allRooms)
}

// GenerateSmallBSPDungeon creates a standard size dungeon using BSP
func (g *DungeonGenerator) GenerateSmallBSPDungeon(mapComp *components.MapComponent) {
	// Call the standard BSP dungeon generation
	g.GenerateBSPDungeon(mapComp)
}
