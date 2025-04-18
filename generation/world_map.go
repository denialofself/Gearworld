package generation

import (
	"fmt"
	"math"
	"math/rand"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// We'll use the biome tile types defined in components package
// No need to redefine them here

// PerlinNoise provides a simple implementation of Perlin noise for terrain generation
type PerlinNoise struct {
	seed        int64
	octaves     int
	scale       float64
	rng         *rand.Rand
	permutation []int
}

// NewPerlinNoise creates a new Perlin noise generator
func NewPerlinNoise(seed int64, octaves int, scale float64) *PerlinNoise {
	p := &PerlinNoise{
		seed:    seed,
		octaves: octaves,
		scale:   scale,
		rng:     rand.New(rand.NewSource(seed)),
	}

	// Initialize permutation table
	p.permutation = make([]int, 256)
	for i := range p.permutation {
		p.permutation[i] = i
	}

	// Shuffle the permutation table
	p.rng.Shuffle(len(p.permutation), func(i, j int) {
		p.permutation[i], p.permutation[j] = p.permutation[j], p.permutation[i]
	})

	return p
}

// Noise2D generates 2D Perlin noise
func (p *PerlinNoise) Noise2D(x, y float64) float64 {
	x = x / p.scale
	y = y / p.scale

	var noise float64
	amplitude := 1.0
	frequency := 1.0
	maxValue := 0.0

	// Sum multiple octaves of noise
	for i := 0; i < p.octaves; i++ {
		noise += p.perlin(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= 0.5
		frequency *= 2.0
	}

	// Normalize
	return noise / maxValue
}

// Helper functions for Perlin noise generation
func (p *PerlinNoise) fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func (p *PerlinNoise) lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

func (p *PerlinNoise) grad(hash int, x, y float64) float64 {
	h := hash & 15

	u := y
	if h < 4 {
		u = x
	}

	v := x
	if h < 12 {
		v = y
	}

	result := u
	if (h & 1) != 0 {
		result = -u
	}

	if (h & 2) != 0 {
		result -= v
	} else {
		result += v
	}

	return result
}

func (p *PerlinNoise) perlin(x, y float64) float64 {
	// Find unit grid cell containing the point
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255

	// Find relative x, y of point in cell
	x -= math.Floor(x)
	y -= math.Floor(y)

	// Compute fade curves
	u := p.fade(x)
	v := p.fade(y)
	// Hash coordinates of the 4 corners
	perm := p.permutation
	// Add blended results from 4 corners of the square
	AA := perm[(perm[X]+Y)&255]
	AB := perm[(perm[X]+Y+1)&255]
	BA := perm[(perm[(X+1)&255]+Y)&255]
	BB := perm[(perm[(X+1)&255]+Y+1)&255]

	return p.lerp(
		p.lerp(p.grad(AA, x, y), p.grad(BA, x-1, y), u),
		p.lerp(p.grad(AB, x, y-1), p.grad(BB, x-1, y-1), u),
		v,
	)
}

// WorldMapGenerator handles procedural generation of the world map
type WorldMapGenerator struct {
	rng      *rand.Rand
	noiseGen *PerlinNoise
}

// NewWorldMapGenerator creates a new world map generator
func NewWorldMapGenerator(seed int64) *WorldMapGenerator {
	return &WorldMapGenerator{
		rng:      rand.New(rand.NewSource(seed)),
		noiseGen: NewPerlinNoise(seed, 6, 20.0),
	}
}

// SetSeed allows setting a specific seed for reproducible world generation
func (g *WorldMapGenerator) SetSeed(seed int64) {
	g.rng = rand.New(rand.NewSource(seed))
	g.noiseGen = NewPerlinNoise(seed, 6, 20.0)
}

// GenerateWorldMap creates a world map using Perlin noise for biome distribution
func (g *WorldMapGenerator) GenerateWorldMap(mapComp *components.MapComponent) {
	// Initialize the map with a basic floor
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			mapComp.SetTile(x, y, components.TileWasteland) // Default is wasteland
		}
	}
	// Generate terrain using Perlin noise
	for y := 0; y < mapComp.Height; y++ {
		for x := 0; x < mapComp.Width; x++ {
			// Get noise value for this position
			elevation := g.noiseGen.Noise2D(float64(x), float64(y))
			moisture := g.noiseGen.Noise2D(float64(x)+500, float64(y)+500) // Offset for different pattern

			// Determine biome based on elevation and moisture only - no railways or substations
			biome := g.determineBiome(elevation, moisture)

			// Debug: Print to console every 10x10 tiles
			if x%10 == 0 && y%10 == 0 {
				fmt.Printf("Tile at %d,%d: Elevation=%.2f, Moisture=%.2f, Biome=%d\n",
					x, y, elevation, moisture, biome)
			}

			mapComp.SetTile(x, y, biome)
		}
	}
}

// determineBiome returns the appropriate biome based on elevation and moisture values
func (g *WorldMapGenerator) determineBiome(elevation, moisture float64) int {
	// Only use the four main biomes: wasteland, desert, dark forest, and mountains
	var biomeType int

	// Use thresholds that ensure a good distribution of biomes based on elevation and moisture
	if elevation > 0.15 {
		biomeType = components.TileMountains // Mountains
	} else if elevation > 0.0 && moisture < 0.0 {
		biomeType = components.TileDesert // Desert
	} else if moisture > 0.1 {
		biomeType = components.TileDarkForest // Dark Forest
	} else {
		biomeType = components.TileWasteland // Default to wasteland
	}

	return biomeType
}

// PlaceCenterSubstation places a single substation at the center of the map
func (g *WorldMapGenerator) PlaceCenterSubstation(mapComp *components.MapComponent) {
	// Calculate center coordinates
	centerX := mapComp.Width / 2
	centerY := mapComp.Height / 2

	// Place the substation
	mapComp.SetTile(centerX, centerY, components.TileSubstation)
	fmt.Printf("Placed substation at center: (%d, %d)\n", centerX, centerY)
}

// PlaceAdditionalSubstations places additional substations at random locations on the map
func (g *WorldMapGenerator) PlaceAdditionalSubstations(mapComp *components.MapComponent, count int) []struct{ x, y int } {
	substations := make([]struct{ x, y int }, 0, count+1) // +1 for center substation

	// Add the center substation first
	centerX := mapComp.Width / 2
	centerY := mapComp.Height / 2
	substations = append(substations, struct{ x, y int }{centerX, centerY})

	// Try to place additional substations
	attempts := 0
	maxAttempts := count * 10 // Limit attempts to avoid infinite loop

	for len(substations) < count+1 && attempts < maxAttempts {
		attempts++
		x := g.rng.Intn(mapComp.Width)
		y := g.rng.Intn(mapComp.Height)

		// Skip if this is a mountain tile
		if mapComp.Tiles[y][x] == components.TileMountains {
			continue
		}

		// Check minimum distance from other substations
		tooClose := false
		for _, s := range substations {
			dx := x - s.x
			dy := y - s.y
			distance := dx*dx + dy*dy
			if distance < 400 { // Minimum distance of 20 tiles squared
				tooClose = true
				break
			}
		}

		if tooClose {
			continue
		}

		// Place the substation
		mapComp.SetTile(x, y, components.TileSubstation)
		substations = append(substations, struct{ x, y int }{x, y})
		fmt.Printf("Placed additional substation at: (%d, %d)\n", x, y)
	}

	return substations
}

// Node represents a position in the pathfinding grid
type Node struct {
	x, y    int
	parent  *Node
	g, h, f float64
}

// NewNode creates a new pathfinding node
func NewNode(x, y int) *Node {
	return &Node{
		x: x,
		y: y,
	}
}

// getNeighbors returns valid neighboring nodes for pathfinding
func (g *WorldMapGenerator) getNeighbors(mapComp *components.MapComponent, node *Node) []*Node {
	neighbors := make([]*Node, 0, 4) // Changed from 8 to 4 for orthogonal only
	directions := [][2]int{
		{0, -1}, // North
		{1, 0},  // East
		{0, 1},  // South
		{-1, 0}, // West
	}

	for _, dir := range directions {
		nx, ny := node.x+dir[0], node.y+dir[1]
		// Check bounds and if the tile is walkable (not a mountain)
		if nx >= 0 && nx < mapComp.Width && ny >= 0 && ny < mapComp.Height &&
			mapComp.Tiles[ny][nx] != components.TileMountains {
			neighbors = append(neighbors, NewNode(nx, ny))
		}
	}

	return neighbors
}

// heuristic calculates the Manhattan distance between two nodes
func heuristic(a, b *Node) float64 {
	dx := float64(a.x - b.x)
	dy := float64(a.y - b.y)
	return math.Abs(dx) + math.Abs(dy)
}

// findPath finds a path from start to end avoiding mountains
func (g *WorldMapGenerator) findPath(mapComp *components.MapComponent, startX, startY, endX, endY int) []*Node {
	start := NewNode(startX, startY)
	end := NewNode(endX, endY)

	openSet := make(map[string]*Node)
	closedSet := make(map[string]*Node)
	openSet[fmt.Sprintf("%d,%d", start.x, start.y)] = start

	for len(openSet) > 0 {
		// Find node with lowest f score
		var current *Node
		lowestF := math.MaxFloat64
		for _, node := range openSet {
			if node.f < lowestF {
				lowestF = node.f
				current = node
			}
		}

		// If we reached the end, reconstruct and return the path
		if current.x == end.x && current.y == end.y {
			path := make([]*Node, 0)
			for current != nil {
				path = append([]*Node{current}, path...)
				current = current.parent
			}
			return path
		}

		// Move current from open to closed set
		delete(openSet, fmt.Sprintf("%d,%d", current.x, current.y))
		closedSet[fmt.Sprintf("%d,%d", current.x, current.y)] = current

		// Check neighbors
		for _, neighbor := range g.getNeighbors(mapComp, current) {
			// Skip if already in closed set
			if _, exists := closedSet[fmt.Sprintf("%d,%d", neighbor.x, neighbor.y)]; exists {
				continue
			}

			// Calculate tentative g score
			tentativeG := current.g + 1.0

			// If neighbor not in open set or found a better path
			if _, exists := openSet[fmt.Sprintf("%d,%d", neighbor.x, neighbor.y)]; !exists || tentativeG < neighbor.g {
				neighbor.parent = current
				neighbor.g = tentativeG
				neighbor.h = heuristic(neighbor, end)
				neighbor.f = neighbor.g + neighbor.h

				if !exists {
					openSet[fmt.Sprintf("%d,%d", neighbor.x, neighbor.y)] = neighbor
				}
			}
		}
	}

	// No path found
	return nil
}

// drawRailwayLine draws a railway line between two points using box drawing characters
func (g *WorldMapGenerator) drawRailwayLine(mapComp *components.MapComponent, x0, y0, x1, y1 int) {
	// Find a path that avoids mountains
	path := g.findPath(mapComp, x0, y0, x1, y1)
	if path == nil {
		return
	}

	// Draw railway along the path
	for i := 0; i < len(path)-1; i++ {
		current := path[i]
		next := path[i+1]

		// Skip if this is a substation
		if mapComp.Tiles[current.y][current.x] == components.TileSubstation {
			continue
		}

		// Check if next tile is a substation
		isNextSubstation := mapComp.Tiles[next.y][next.x] == components.TileSubstation

		// Check connections in all four directions
		connTop := current.y > 0 && (mapComp.Tiles[current.y-1][current.x] == components.TileRailwayVertical ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayTopLeft ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayTopRight ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayTeeLeft ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayTeeRight ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayTeeBottom ||
			mapComp.Tiles[current.y-1][current.x] == components.TileRailwayCross ||
			(isNextSubstation && next.y == current.y-1))

		connRight := current.x < mapComp.Width-1 && (mapComp.Tiles[current.y][current.x+1] == components.TileRailwayHorizontal ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayTopRight ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayBottomRight ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayTeeLeft ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayTeeTop ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayTeeBottom ||
			mapComp.Tiles[current.y][current.x+1] == components.TileRailwayCross ||
			(isNextSubstation && next.x == current.x+1))

		connBottom := current.y < mapComp.Height-1 && (mapComp.Tiles[current.y+1][current.x] == components.TileRailwayVertical ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayBottomLeft ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayBottomRight ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayTeeLeft ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayTeeRight ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayTeeTop ||
			mapComp.Tiles[current.y+1][current.x] == components.TileRailwayCross ||
			(isNextSubstation && next.y == current.y+1))

		connLeft := current.x > 0 && (mapComp.Tiles[current.y][current.x-1] == components.TileRailwayHorizontal ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayTopLeft ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayBottomLeft ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayTeeRight ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayTeeTop ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayTeeBottom ||
			mapComp.Tiles[current.y][current.x-1] == components.TileRailwayCross ||
			(isNextSubstation && next.x == current.x-1))

		// Determine direction of movement
		dx := next.x - current.x
		dy := next.y - current.y

		// Determine the appropriate railway tile type based on connections and movement
		var tileType int
		switch {
		case dx > 0: // Moving right
			if connTop && connBottom {
				tileType = components.TileRailwayTeeLeft
			} else if connTop {
				tileType = components.TileRailwayBottomLeft
			} else if connBottom {
				tileType = components.TileRailwayTopLeft
			} else {
				tileType = components.TileRailwayHorizontal
			}
		case dx < 0: // Moving left
			if connTop && connBottom {
				tileType = components.TileRailwayTeeRight
			} else if connTop {
				tileType = components.TileRailwayBottomRight
			} else if connBottom {
				tileType = components.TileRailwayTopRight
			} else {
				tileType = components.TileRailwayHorizontal
			}
		case dy > 0: // Moving down
			if connLeft && connRight {
				tileType = components.TileRailwayTeeTop
			} else if connLeft {
				tileType = components.TileRailwayTopRight
			} else if connRight {
				tileType = components.TileRailwayTopLeft
			} else {
				tileType = components.TileRailwayVertical
			}
		case dy < 0: // Moving up
			if connLeft && connRight {
				tileType = components.TileRailwayTeeBottom
			} else if connLeft {
				tileType = components.TileRailwayBottomRight
			} else if connRight {
				tileType = components.TileRailwayBottomLeft
			} else {
				tileType = components.TileRailwayVertical
			}
		}

		// Set the tile type
		mapComp.SetTile(current.x, current.y, tileType)
	}
}

// CreateWorldMapEntity creates a new entity with a MapComponent for the world map
func (g *WorldMapGenerator) CreateWorldMapEntity(world *ecs.World, width, height int) *ecs.Entity {
	// Create the map entity
	mapEntity := world.CreateEntity()
	mapEntity.AddTag("worldmap")
	world.TagEntity(mapEntity.ID, "worldmap")

	// Create map component
	mapComp := components.NewMapComponent(width, height)
	world.AddComponent(mapEntity.ID, components.MapComponentID, mapComp)

	// Generate the world map
	g.GenerateWorldMap(mapComp)

	// Calculate center coordinates
	centerX := mapComp.Width / 2
	centerY := mapComp.Height / 2

	// Place the center substation
	mapComp.SetTile(centerX, centerY, components.TileSubstation)
	fmt.Printf("Placed center substation at: (%d, %d)\n", centerX, centerY)

	// Place the alpha station near the center
	alphaStation := struct{ x, y int }{}
	attempts := 0
	maxAttempts := 20
	minDistance := 5  // Minimum distance from center
	maxDistance := 15 // Maximum distance from center

	for attempts < maxAttempts {
		// Generate a random angle
		angle := g.rng.Float64() * 2 * math.Pi
		// Generate a random distance between min and max
		distance := minDistance + g.rng.Intn(maxDistance-minDistance)

		// Calculate position
		alphaX := centerX + int(float64(distance)*math.Cos(angle))
		alphaY := centerY + int(float64(distance)*math.Sin(angle))

		// Check bounds and if the tile is walkable
		if alphaX >= 0 && alphaX < mapComp.Width &&
			alphaY >= 0 && alphaY < mapComp.Height &&
			mapComp.Tiles[alphaY][alphaX] != components.TileMountains {
			alphaStation.x = alphaX
			alphaStation.y = alphaY
			mapComp.SetTile(alphaX, alphaY, components.TileSubstation)
			fmt.Printf("Placed alpha station at: (%d, %d)\n", alphaX, alphaY)
			break
		}
		attempts++
	}

	// Place additional stations (8 total)
	additionalStations := make([]struct{ x, y int }, 0, 8)
	attempts = 0
	maxAttempts = 100 // Increased attempts for more stations

	for len(additionalStations) < 8 && attempts < maxAttempts {
		attempts++
		x := g.rng.Intn(mapComp.Width)
		y := g.rng.Intn(mapComp.Height)

		// Skip if this is a mountain tile
		if mapComp.Tiles[y][x] == components.TileMountains {
			continue
		}

		// Check minimum distance from other stations
		tooClose := false
		for _, s := range additionalStations {
			dx := x - s.x
			dy := y - s.y
			distance := dx*dx + dy*dy
			if distance < 400 { // Minimum distance of 20 tiles squared
				tooClose = true
				break
			}
		}

		// Also check distance from center and alpha
		dx := x - centerX
		dy := y - centerY
		if dx*dx+dy*dy < 400 {
			tooClose = true
		}
		dx = x - alphaStation.x
		dy = y - alphaStation.y
		if dx*dx+dy*dy < 400 {
			tooClose = true
		}

		if tooClose {
			continue
		}

		// Place the station
		mapComp.SetTile(x, y, components.TileSubstation)
		additionalStations = append(additionalStations, struct{ x, y int }{x, y})
		fmt.Printf("Placed additional station at: (%d, %d)\n", x, y)
	}

	// Connect center to alpha station with broken railway
	path := g.findPath(mapComp, centerX, centerY, alphaStation.x, alphaStation.y)
	if path != nil {
		// Randomly remove two tiles from the path
		if len(path) > 2 {
			removeIndices := make(map[int]bool)
			for len(removeIndices) < 2 {
				idx := g.rng.Intn(len(path)-2) + 1 // Don't remove first or last tile
				removeIndices[idx] = true
			}

			// Draw the railway with gaps
			for i := 0; i < len(path)-1; i++ {
				if !removeIndices[i] {
					current := path[i]
					next := path[i+1]
					g.drawRailwayLine(mapComp, current.x, current.y, next.x, next.y)
				}
			}
		}
	}

	return mapEntity
}
