package generation

import (
	"ebiten-rogue/components"
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

// CorridorStart and CorridorEnd for the BSPNode
func (node *BSPNode) CorridorStart() []int {
	return node.corridorStart
}

func (node *BSPNode) CorridorEnd() []int {
	return node.corridorEnd
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

	// Verify room connectivity and fix orphaned rooms
	g.ensureRoomConnectivity(root, mapComp)

	// Add features (water, lava, stairs, trees, etc.)
	var rooms [][4]int
	g.collectRooms(root, &rooms)
	g.AddFeatures(mapComp, rooms)

	// Apply box drawing characters to the walls
	g.applyImprovedBoxDrawingWalls(mapComp)
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

// GenerateSmallBSPDungeon creates a standard size dungeon using BSP
func (g *DungeonGenerator) GenerateSmallBSPDungeon(mapComp *components.MapComponent) {
	// Call the standard BSP dungeon generation
	g.GenerateBSPDungeon(mapComp)
}

// GenerateLargeBSPDungeon creates a 10x10 screen large dungeon using BSP
func (g *DungeonGenerator) GenerateLargeBSPDungeon(mapComp *components.MapComponent) {
	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	// Create the root node for the entire dungeon
	root := &BSPNode{
		X:      0,
		Y:      0,
		Width:  mapComp.Width,
		Height: mapComp.Height,
	}

	// First split: divide the dungeon into macro sections (3x3 grid)
	macroSections := g.createMacroSections(root, 3, 3)

	// For each macro section, apply BSP partitioning
	var allRooms [][4]int
	for _, section := range macroSections {
		// Split this section recursively (max depth 4)
		g.splitNodeImproved(section, 0, 4)

		// Create rooms in leaf nodes
		g.createRoomsInNode(section)

		// Connect rooms within this section
		g.connectRoomsInNode(section)

		// Draw rooms to the map
		g.drawRooms(section, mapComp)

		// Collect room information for features
		g.collectRoomsFromNode(section, &allRooms)
	}

	// Connect adjacent macro sections
	g.connectMacroSections(macroSections, mapComp)

	// Verify connectivity and fix orphaned rooms
	g.verifyGlobalConnectivity(mapComp)

	// Add features to the dungeon
	g.AddFeatures(mapComp, allRooms)

	// Apply box drawing characters to all walls
	g.applyWallTypes(mapComp)
}

// createMacroSections divides the dungeon into grid of sections
func (g *DungeonGenerator) createMacroSections(root *BSPNode, gridWidth, gridHeight int) []*BSPNode {
	sectionWidth := root.Width / gridWidth
	sectionHeight := root.Height / gridHeight

	var sections []*BSPNode

	for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			section := &BSPNode{
				X:      root.X + x*sectionWidth,
				Y:      root.Y + y*sectionHeight,
				Width:  sectionWidth,
				Height: sectionHeight,
			}
			sections = append(sections, section)
		}
	}

	return sections
}

// splitNodeImproved is an improved version of the BSP node splitting algorithm
func (g *DungeonGenerator) splitNodeImproved(node *BSPNode, depth, maxDepth int) {
	// Stop recursion if we've reached maximum depth
	if depth >= maxDepth {
		return
	}

	// Minimum allowable room size
	minRoomSize := 8

	// Check if node is too small to split further
	if node.Width < minRoomSize*2 || node.Height < minRoomSize*2 {
		return
	}

	// Decide split direction based on aspect ratio
	horizontal := false
	if float64(node.Width) > float64(node.Height)*1.25 {
		horizontal = false // Split vertically when wider
	} else if float64(node.Height) > float64(node.Width)*1.25 {
		horizontal = true // Split horizontally when taller
	} else {
		// Random split direction otherwise
		horizontal = g.rng.Intn(2) == 0
	}

	// Calculate split position with a bit of randomness
	var splitPos int
	minSplitRatio := 0.3 // Ensure neither side is too small (at least 30%)
	maxSplitRatio := 0.7 // Ensure neither side is too large (at most 70%)

	if horizontal {
		// Horizontal split (creates top and bottom)
		minSplit := int(float64(node.Height) * minSplitRatio)
		maxSplit := int(float64(node.Height) * maxSplitRatio)
		splitRange := maxSplit - minSplit

		if splitRange <= 0 {
			splitPos = minSplit
		} else {
			splitPos = minSplit + g.rng.Intn(splitRange)
		}

		// Create children
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
		// Vertical split (creates left and right)
		minSplit := int(float64(node.Width) * minSplitRatio)
		maxSplit := int(float64(node.Width) * maxSplitRatio)
		splitRange := maxSplit - minSplit

		if splitRange <= 0 {
			splitPos = minSplit
		} else {
			splitPos = minSplit + g.rng.Intn(splitRange)
		}

		// Create children
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

	// Continue recursively splitting children
	g.splitNodeImproved(node.Left, depth+1, maxDepth)
	g.splitNodeImproved(node.Right, depth+1, maxDepth)
}

// createRoomsInNode creates rooms in all leaf nodes of the given subtree
func (g *DungeonGenerator) createRoomsInNode(node *BSPNode) {
	// If this is a leaf node (no children), create a room
	if node.Left == nil && node.Right == nil {
		// Room size variation: 60-90% of node size
		roomWidthRatio := 0.6 + g.rng.Float64()*0.3
		roomHeightRatio := 0.6 + g.rng.Float64()*0.3

		roomWidth := max(5, int(float64(node.Width)*roomWidthRatio))
		roomHeight := max(5, int(float64(node.Height)*roomHeightRatio))

		// Position room with random padding (but centered in node)
		roomX := node.X + (node.Width-roomWidth)/2
		roomY := node.Y + (node.Height-roomHeight)/2

		// Add a bit of random offset
		roomX += g.rng.Intn(3) - 1
		roomY += g.rng.Intn(3) - 1

		// Ensure room is within node boundaries
		roomX = max(node.X+1, min(roomX, node.X+node.Width-roomWidth-1))
		roomY = max(node.Y+1, min(roomY, node.Y+node.Height-roomHeight-1))

		// Create and store the room
		node.Room = &Room{
			X:      roomX,
			Y:      roomY,
			Width:  roomWidth,
			Height: roomHeight,
		}
		return
	}

	// Continue with children if not a leaf
	if node.Left != nil {
		g.createRoomsInNode(node.Left)
	}
	if node.Right != nil {
		g.createRoomsInNode(node.Right)
	}
}

// connectRoomsInNode connects all rooms in the BSP tree
func (g *DungeonGenerator) connectRoomsInNode(node *BSPNode) {
	// If this node has two child nodes, connect rooms between them
	if node.Left != nil && node.Right != nil {
		leftRoom := g.findRoomInNode(node.Left)
		rightRoom := g.findRoomInNode(node.Right)

		if leftRoom != nil && rightRoom != nil {
			// Get centers of both rooms
			x1 := leftRoom.X + leftRoom.Width/2
			y1 := leftRoom.Y + leftRoom.Height/2
			x2 := rightRoom.X + rightRoom.Width/2
			y2 := rightRoom.Y + rightRoom.Height/2

			// Store corridor information for later drawing
			node.corridorStart = []int{x1, y1}
			node.corridorEnd = []int{x2, y2}
		}
	}

	// Process children recursively
	if node.Left != nil {
		g.connectRoomsInNode(node.Left)
	}
	if node.Right != nil {
		g.connectRoomsInNode(node.Right)
	}
}

// findRoomInNode finds a room in the subtree rooted at node
func (g *DungeonGenerator) findRoomInNode(node *BSPNode) *Room {
	// If this node has a room, return it
	if node.Room != nil {
		return node.Room
	}

	// Otherwise search in children
	var room *Room
	if node.Left != nil {
		room = g.findRoomInNode(node.Left)
		if room != nil {
			return room
		}
	}
	if node.Right != nil {
		room = g.findRoomInNode(node.Right)
		if room != nil {
			return room
		}
	}

	return nil
}

// drawRooms draws all rooms and corridors in the BSP tree
func (g *DungeonGenerator) drawRooms(node *BSPNode, mapComp *components.MapComponent) {
	// Draw the room if present
	if node.Room != nil {
		for y := node.Room.Y; y < node.Room.Y+node.Room.Height; y++ {
			for x := node.Room.X; x < node.Room.X+node.Room.Width; x++ {
				if x >= 0 && x < mapComp.Width && y >= 0 && y < mapComp.Height {
					mapComp.SetTile(x, y, components.TileFloor)

					// Occasionally add grass for variety (5% chance)
					if g.rng.Intn(100) < 5 {
						mapComp.SetTile(x, y, components.TileGrass)
					}
				}
			}
		}
	}

	// Draw corridor if this node has one
	if node.corridorStart != nil && node.corridorEnd != nil {
		x1, y1 := node.corridorStart[0], node.corridorStart[1]
		x2, y2 := node.corridorEnd[0], node.corridorEnd[1]

		// Use L-shaped corridors (more reliable than direct corridors)
		// Randomly decide whether to go horizontal-then-vertical or vertical-then-horizontal
		if g.rng.Intn(2) == 0 {
			// Horizontal then vertical
			g.createHorizontalCorridor(mapComp, x1, x2, y1)
			g.createVerticalCorridor(mapComp, y1, y2, x2)
		} else {
			// Vertical then horizontal
			g.createVerticalCorridor(mapComp, y1, y2, x1)
			g.createHorizontalCorridor(mapComp, x1, x2, y2)
		}

		// Add doors occasionally (20% chance at corridor junctions)
		if g.rng.Intn(100) < 20 {
			doorX, doorY := x1, y1
			if g.rng.Intn(2) == 0 {
				doorX, doorY = x2, y2
			}
			// Make sure position is valid and is a corridor (floor)
			if doorX >= 0 && doorX < mapComp.Width && doorY >= 0 && doorY < mapComp.Height {
				if mapComp.Tiles[doorY][doorX] == components.TileFloor {
					mapComp.SetTile(doorX, doorY, components.TileDoor)
				}
			}
		}
	}

	// Recursively draw child nodes
	if node.Left != nil {
		g.drawRooms(node.Left, mapComp)
	}
	if node.Right != nil {
		g.drawRooms(node.Right, mapComp)
	}
}

// collectRoomsFromNode gathers all rooms from the BSP tree into a slice
func (g *DungeonGenerator) collectRoomsFromNode(node *BSPNode, rooms *[][4]int) {
	if node.Room != nil {
		*rooms = append(*rooms, [4]int{
			node.Room.X,
			node.Room.Y,
			node.Room.Width,
			node.Room.Height,
		})
	}

	if node.Left != nil {
		g.collectRoomsFromNode(node.Left, rooms)
	}
	if node.Right != nil {
		g.collectRoomsFromNode(node.Right, rooms)
	}
}

// connectMacroSections connects the different macro sections of the dungeon
func (g *DungeonGenerator) connectMacroSections(sections []*BSPNode, mapComp *components.MapComponent) {
	// Ensure each section connects to at least one adjacent section
	gridWidth, gridHeight := 3, 3 // We're using a 3x3 grid

	// For each section, find a room and connect to an adjacent section's room
	for i, section := range sections {
		// Calculate grid position
		gridX := i % gridWidth
		gridY := i / gridWidth

		// Find a room in this section
		room1 := g.findRoomInNode(section)
		if room1 == nil {
			continue
		}

		// Get room center
		x1 := room1.X + room1.Width/2
		y1 := room1.Y + room1.Height/2

		// Try to connect to right section
		if gridX < gridWidth-1 && i+1 < len(sections) {
			rightSection := sections[i+1]
			if room2 := g.findRoomInNode(rightSection); room2 != nil {
				// Connect to a room in the right section
				x2 := room2.X + room2.Width/2
				y2 := room2.Y + room2.Height/2

				// Use L-shaped corridor
				g.createHorizontalCorridor(mapComp, x1, x2, y1)
				g.createVerticalCorridor(mapComp, y1, y2, x2)
			}
		}

		// Try to connect to section below
		if gridY < gridHeight-1 && i+gridWidth < len(sections) {
			belowSection := sections[i+gridWidth]
			if room2 := g.findRoomInNode(belowSection); room2 != nil {
				// Connect to a room in the section below
				x2 := room2.X + room2.Width/2
				y2 := room2.Y + room2.Height/2

				// Use L-shaped corridor
				g.createVerticalCorridor(mapComp, y1, y2, x1)
				g.createHorizontalCorridor(mapComp, x1, x2, y2)
			}
		}
	}
}

// verifyGlobalConnectivity ensures all floor tiles are connected
func (g *DungeonGenerator) verifyGlobalConnectivity(mapComp *components.MapComponent) {
	// Create connectivity map
	visited := make([][]bool, mapComp.Height)
	for i := range visited {
		visited[i] = make([]bool, mapComp.Width)
	}

	// Find a starting floor tile
	var startX, startY int
	foundStart := false
	for y := 0; y < mapComp.Height && !foundStart; y++ {
		for x := 0; x < mapComp.Width && !foundStart; x++ {
			if mapComp.Tiles[y][x] == components.TileFloor {
				startX, startY = x, y
				foundStart = true
			}
		}
	}

	if !foundStart {
		return // No floor tiles found at all
	}

	// Perform flood fill from starting point
	g.floodFillConnectivity(mapComp, startX, startY, visited)

	// Find disconnected regions and connect them
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if (mapComp.Tiles[y][x] == components.TileFloor ||
				mapComp.Tiles[y][x] == components.TileGrass) && !visited[y][x] {
				// Found disconnected region, connect to main dungeon
				g.connectToMainDungeon(mapComp, x, y, visited)

				// Flood fill from this point to mark its region as visited
				g.floodFillConnectivity(mapComp, x, y, visited)
			}
		}
	}
}

// floodFillConnectivity marks all connected floor tiles as visited
func (g *DungeonGenerator) floodFillConnectivity(mapComp *components.MapComponent, x, y int, visited [][]bool) {
	// Simple BFS flood fill
	queue := [][2]int{{x, y}}
	visited[y][x] = true

	// Four principal directions
	dirs := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, dir := range dirs {
			nx, ny := curr[0]+dir[0], curr[1]+dir[1]

			// Check bounds
			if nx < 0 || nx >= mapComp.Width || ny < 0 || ny >= mapComp.Height {
				continue
			}

			// Check if this is a walkable tile and not visited
			if !visited[ny][nx] && isWalkable(mapComp.Tiles[ny][nx]) {
				visited[ny][nx] = true
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}
}

// isWalkable checks if a tile can be walked on
func isWalkable(tileType int) bool {
	return tileType == components.TileFloor ||
		tileType == components.TileGrass ||
		tileType == components.TileDoor
}

// connectToMainDungeon connects a disconnected region to the main dungeon
func (g *DungeonGenerator) connectToMainDungeon(mapComp *components.MapComponent, x, y int, visited [][]bool) {
	// Find the closest floor tile that is part of the main dungeon
	minDist := mapComp.Width + mapComp.Height // max possible distance
	var targetX, targetY int

	for cy := 0; cy < mapComp.Height; cy++ {
		for cx := 0; cx < mapComp.Width; cx++ {
			if visited[cy][cx] && isWalkable(mapComp.Tiles[cy][cx]) {
				dist := abs(cx-x) + abs(cy-y)
				if dist < minDist {
					minDist = dist
					targetX, targetY = cx, cy
				}
			}
		}
	}

	// Create a corridor to connect them
	g.CreateCorridor(mapComp, x, y, targetX, targetY)
}

// applyWallTypes sets appropriate wall types for all walls in the dungeon
func (g *DungeonGenerator) applyWallTypes(mapComp *components.MapComponent) {
	// First pass: set all walls back to basic wall type to ensure clean processing
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if IsAnyWallType(mapComp.Tiles[y][x]) {
				mapComp.Tiles[y][x] = components.TileWall
			}
		}
	}

	// Second pass: mark all walls adjacent to non-wall tiles
	wallTiles := make([][]bool, mapComp.Height)
	for y := range wallTiles {
		wallTiles[y] = make([]bool, mapComp.Width)
		for x := 0; x < mapComp.Width; x++ {
			if mapComp.Tiles[y][x] == components.TileWall {
				// Check if this wall has at least one adjacent non-wall tile
				if g.hasAdjacentNonWall(mapComp, x, y) {
					wallTiles[y][x] = true
				}
			}
		}
	}

	// Third pass: calculate and apply wall connection masks
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			if wallTiles[y][x] {
				mask := g.calculateWallConnectionMask(mapComp, x, y)
				if mask >= 0 && mask <= 15 { // Ensure mask is valid
					mapComp.Tiles[y][x] = WallTileLookup[mask]
				}
			}
		}
	}
}

// hasAdjacentNonWall checks if a tile has any adjacent non-wall tiles
func (g *DungeonGenerator) hasAdjacentNonWall(mapComp *components.MapComponent, x, y int) bool {
	// Check the four cardinal directions
	// Note: We consider ALL walls, not just basic walls
	if y > 0 && !IsAnyWallType(mapComp.Tiles[y-1][x]) {
		return true // Has floor/door/etc above
	}
	if x < mapComp.Width-1 && !IsAnyWallType(mapComp.Tiles[y][x+1]) {
		return true // Has floor/door/etc to the right
	}
	if y < mapComp.Height-1 && !IsAnyWallType(mapComp.Tiles[y+1][x]) {
		return true // Has floor/door/etc below
	}
	if x > 0 && !IsAnyWallType(mapComp.Tiles[y][x-1]) {
		return true // Has floor/door/etc to the left
	}
	return false
}

// calculateWallConnectionMask determines how a wall should connect to adjacent walls
func (g *DungeonGenerator) calculateWallConnectionMask(mapComp *components.MapComponent, x, y int) int {
	mask := 0

	// Check in which directions this wall connects to other walls
	// We're treating out-of-bounds as wall tiles
	// We also treat ALL wall types as connected walls
	if y == 0 || (y > 0 && IsAnyWallType(mapComp.Tiles[y-1][x])) {
		mask |= WallConnectTop
	}
	if x == mapComp.Width-1 || (x < mapComp.Width-1 && IsAnyWallType(mapComp.Tiles[y][x+1])) {
		mask |= WallConnectRight
	}
	if y == mapComp.Height-1 || (y < mapComp.Height-1 && IsAnyWallType(mapComp.Tiles[y+1][x])) {
		mask |= WallConnectBottom
	}
	if x == 0 || (x > 0 && IsAnyWallType(mapComp.Tiles[y][x-1])) {
		mask |= WallConnectLeft
	}

	return mask
}

// GenerateGiantBSPDungeon creates a 20x20 screen massive dungeon using BSP
func (g *DungeonGenerator) GenerateGiantBSPDungeon(mapComp *components.MapComponent) {
	// Fill the map with walls initially
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWall)
		}
	}

	// Calculate section dimensions (20x20 grid)
	sectionWidth := mapComp.Width / 20
	sectionHeight := mapComp.Height / 20

	// Store all section nodes for connectivity checking
	var allSectionNodes []*BSPNode

	// Generate each section
	for gridY := 0; gridY < 20; gridY++ {
		for gridX := 0; gridX < 20; gridX++ {
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
			allSectionNodes = append(allSectionNodes, sectionNode)

			// Split this section with lower depth for smaller sections
			g.splitNode(sectionNode, 0, 2)

			// Create and connect rooms
			g.createRoomsInLeaves(sectionNode)
			g.connectRooms(sectionNode)
			g.drawBSPDungeon(sectionNode, mapComp)

			// Connect to neighboring sections
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

				// Connect to either left or above neighbor
				if gridX > 0 && gridY > 0 {
					// Connect to both previous sections occasionally
					if g.rng.Intn(100) < 30 { // 30% chance to connect to both
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

	// Additional connectivity pass over the entire dungeon to ensure all sections are connected
	for i := 0; i < len(allSectionNodes); i++ {
		g.ensureRoomConnectivity(allSectionNodes[i], mapComp)
	}

	// Add features to the dungeon
	var allRooms [][4]int
	for _, node := range allSectionNodes {
		g.collectRooms(node, &allRooms)
	}
	g.AddFeatures(mapComp, allRooms)

	// Apply improved box drawing characters to the walls
	g.applyImprovedBoxDrawingWalls(mapComp)
}

// ensureRoomConnectivity verifies that all rooms in the dungeon are connected
// and fixes any orphaned rooms by creating additional corridors
func (g *DungeonGenerator) ensureRoomConnectivity(rootNode *BSPNode, mapComp *components.MapComponent) {
	// Collect all rooms
	var rooms [][4]int
	g.collectRooms(rootNode, &rooms)
	if len(rooms) <= 1 {
		return // No need to check if there's only one room
	}

	// Build a graph representation of room connectivity
	connected := make([]bool, len(rooms))
	connected[0] = true // Start with first room as connected

	// Keep track of which rooms we've checked
	changed := true
	for changed {
		changed = false

		// For each connected room, check if it connects to any unconnected rooms
		for i := 0; i < len(rooms); i++ {
			if !connected[i] {
				continue
			}

			// This is a connected room, check if it connects to any unconnected rooms
			room1 := rooms[i]
			r1x, r1y := room1[0]+room1[2]/2, room1[1]+room1[3]/2 // center

			for j := 0; j < len(rooms); j++ {
				if connected[j] {
					continue // already connected
				}

				// Check if there's a path between these rooms
				room2 := rooms[j]
				r2x, r2y := room2[0]+room2[2]/2, room2[1]+room2[3]/2 // center

				if g.roomsAreConnected(mapComp, r1x, r1y, r2x, r2y) {
					connected[j] = true
					changed = true
				}
			}
		}
	}

	// Check for any unconnected rooms and connect them
	for i := 0; i < len(rooms); i++ {
		if !connected[i] {
			// This room is not connected, find the closest connected room
			closestRoom := -1
			minDistance := mapComp.Width * mapComp.Height // max possible distance

			unconnectedRoom := rooms[i]
			ur_x, ur_y := unconnectedRoom[0]+unconnectedRoom[2]/2, unconnectedRoom[1]+unconnectedRoom[3]/2

			for j := 0; j < len(rooms); j++ {
				if !connected[j] || i == j {
					continue
				}

				connectedRoom := rooms[j]
				cr_x, cr_y := connectedRoom[0]+connectedRoom[2]/2, connectedRoom[1]+connectedRoom[3]/2

				// Use Manhattan distance
				distance := abs(ur_x-cr_x) + abs(ur_y-cr_y)
				if distance < minDistance {
					minDistance = distance
					closestRoom = j
				}
			}

			if closestRoom != -1 {
				// Connect this room to the closest connected room
				cr := rooms[closestRoom]
				cr_x, cr_y := cr[0]+cr[2]/2, cr[1]+cr[3]/2

				// Create a corridor between the centers
				g.CreateCorridor(mapComp, ur_x, ur_y, cr_x, cr_y)
				connected[i] = true
			}
		}
	}
}

// roomsAreConnected checks if there is a valid path between two room centers
func (g *DungeonGenerator) roomsAreConnected(mapComp *components.MapComponent, x1, y1, x2, y2 int) bool {
	// Breadth-first search to find a path
	queue := [][2]int{{x1, y1}}
	visited := make(map[int]bool)

	// Mark starting position as visited
	visited[y1*mapComp.Width+x1] = true

	// Define the four directions we can move
	directions := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

	// BFS loop
	for len(queue) > 0 {
		// Get the next position
		pos := queue[0]
		queue = queue[1:]

		x, y := pos[0], pos[1]

		// Check if we've reached the destination
		if x == x2 && y == y2 {
			return true
		}

		// Try each direction
		for _, dir := range directions {
			nx, ny := x+dir[0], y+dir[1]

			// Check if the new position is valid
			if nx < 0 || nx >= mapComp.Width || ny < 0 || ny >= mapComp.Height {
				continue
			}

			// Check if it's a floor tile and not visited
			if mapComp.Tiles[ny][nx] != components.TileWall && !visited[ny*mapComp.Width+nx] {
				queue = append(queue, [2]int{nx, ny})
				visited[ny*mapComp.Width+nx] = true
			}
		}
	}

	// If we get here, no path was found
	return false
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// applyImprovedBoxDrawingWalls is an enhanced version of the box drawing wall application
// that ensures all walls, including interior ones, are properly rendered with appropriate characters
func (g *DungeonGenerator) applyImprovedBoxDrawingWalls(mapComp *components.MapComponent) { // Phase 1: Apply initial wall types to all walls
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			// Only process basic wall tiles
			if IsWallTile(mapComp.Tiles[y][x]) {
				// Calculate the appropriate wall mask
				maskValue := CalculateWallMask(mapComp, x, y)
				// Convert to appropriate wall type based on connections
				mapComp.SetTile(x, y, WallTileLookup[maskValue])
			}
		}
	}

	// Phase 2: Fix inconsistencies between walls
	// This makes multiple passes to resolve cases where walls have incompatible neighbors
	const maxPasses = 3
	for pass := 0; pass < maxPasses; pass++ {
		changes := false
		for y := 1; y < mapComp.Height-1; y++ {
			for x := 1; x < mapComp.Width-1; x++ {
				// Only process wall tiles
				if IsAnyWallType(mapComp.Tiles[y][x]) {
					changes = g.fixWallInconsistencies(mapComp, x, y) || changes
				}
			}
		}

		// If no changes were made, we're done
		if !changes {
			break
		}
	}
}

// fixWallInconsistencies checks and fixes inconsistencies between adjacent wall tiles
func (g *DungeonGenerator) fixWallInconsistencies(mapComp *components.MapComponent, x, y int) bool {
	// Get the current wall type
	currentType := mapComp.Tiles[y][x]

	// Check if surrounding wall types match current wall's connection points
	// This looks for special cases where wall connections are inconsistent

	// Check wall above
	if y > 0 && IsAnyWallType(mapComp.Tiles[y-1][x]) {
		// If current wall doesn't connect upward but the wall above connects downward
		if !wallConnectsUp(currentType) && wallConnectsDown(mapComp.Tiles[y-1][x]) {
			// Recalculate with forced connection upward
			maskValue := CalculateWallMask(mapComp, x, y) | WallConnectTop
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
		// If current wall connects upward but wall above doesn't connect downward
		if wallConnectsUp(currentType) && !wallConnectsDown(mapComp.Tiles[y-1][x]) {
			// Recalculate with forced disconnect upward
			maskValue := CalculateWallMask(mapComp, x, y) & ^WallConnectTop
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
	}

	// Check wall to the right
	if x < mapComp.Width-1 && IsAnyWallType(mapComp.Tiles[y][x+1]) {
		// If current wall doesn't connect right but the wall to the right connects left
		if !wallConnectsRight(currentType) && wallConnectsLeft(mapComp.Tiles[y][x+1]) {
			maskValue := CalculateWallMask(mapComp, x, y) | WallConnectRight
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
		// If current wall connects right but wall to the right doesn't connect left
		if wallConnectsRight(currentType) && !wallConnectsLeft(mapComp.Tiles[y][x+1]) {
			maskValue := CalculateWallMask(mapComp, x, y) & ^WallConnectRight
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
	}

	// Check wall below
	if y < mapComp.Height-1 && IsAnyWallType(mapComp.Tiles[y+1][x]) {
		// If current wall doesn't connect down but the wall below connects up
		if !wallConnectsDown(currentType) && wallConnectsUp(mapComp.Tiles[y+1][x]) {
			maskValue := CalculateWallMask(mapComp, x, y) | WallConnectBottom
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
		// If current wall connects down but wall below doesn't connect up
		if wallConnectsDown(currentType) && !wallConnectsUp(mapComp.Tiles[y+1][x]) {
			maskValue := CalculateWallMask(mapComp, x, y) & ^WallConnectBottom
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
	}

	// Check wall to the left
	if x > 0 && IsAnyWallType(mapComp.Tiles[y][x-1]) {
		// If current wall doesn't connect left but the wall to the left connects right
		if !wallConnectsLeft(currentType) && wallConnectsRight(mapComp.Tiles[y][x-1]) {
			maskValue := CalculateWallMask(mapComp, x, y) | WallConnectLeft
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}

		// If current wall connects left but wall to the left doesn't connect right
		if wallConnectsLeft(currentType) && !wallConnectsRight(mapComp.Tiles[y][x-1]) {
			maskValue := CalculateWallMask(mapComp, x, y) & ^WallConnectLeft
			mapComp.SetTile(x, y, WallTileLookup[maskValue])
			return true
		}
	}

	return false
}

// Helper functions to check wall connections in different directions
func wallConnectsUp(wallType int) bool {
	return wallType == components.TileWallVertical || // 10 │
		wallType == components.TileWallTeeLeft || // 15 ├
		wallType == components.TileWallTeeRight || // 16 ┤
		wallType == components.TileWallCross || // 19 ┼
		wallType == components.TileWallBottomLeft || // 13 └
		wallType == components.TileWallBottomRight // 14 ┘
}

func wallConnectsRight(wallType int) bool {
	return wallType == components.TileWallHorizontal || // 9 ─
		wallType == components.TileWallTeeTop || // 17 ┬
		wallType == components.TileWallTeeBottom || // 18 ┴
		wallType == components.TileWallCross || // 19 ┼
		wallType == components.TileWallTopLeft || // 11 ┌
		wallType == components.TileWallBottomLeft // 13 └
}

func wallConnectsDown(wallType int) bool {
	return wallType == components.TileWallVertical || // 10 │
		wallType == components.TileWallTeeLeft || // 15 ├
		wallType == components.TileWallTeeRight || // 16 ┤
		wallType == components.TileWallCross || // 19 ┼
		wallType == components.TileWallTopLeft || // 11 ┌
		wallType == components.TileWallTopRight // 12 ┐
}

func wallConnectsLeft(wallType int) bool {
	return wallType == components.TileWallHorizontal || // 9 ─
		wallType == components.TileWallTeeTop || // 17 ┬
		wallType == components.TileWallTeeBottom || // 18 ┴
		wallType == components.TileWallCross || // 19 ┼
		wallType == components.TileWallTopRight || // 12 ┐
		wallType == components.TileWallBottomRight // 14 ┘
}
