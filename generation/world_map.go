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

	// Place a single substation at the center of the map
	g.PlaceCenterSubstation(mapComp)

	return mapEntity
}

// No railway generation functions needed anymore

// No additional station placement functions needed with simplified approach

// No railway drawing functions needed anymore
