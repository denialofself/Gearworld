package generation

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// MapGenerator defines the interface for map generation functionality
// This interface allows us to break the import cycle between systems and generation packages
type MapGenerator interface {
	GenerateSmallBSPDungeon(world *ecs.World, width, height int) *ecs.Entity
	GenerateLargeBSPDungeon(world *ecs.World, width, height int) *ecs.Entity
	FindEmptyPosition(mapComp *components.MapComponent) (int, int)
}
