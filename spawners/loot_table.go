package spawners

import (
	"ebiten-rogue/ecs"
	"math/rand"
)

// LootTable defines a table of possible items and their drop chances
type LootTable struct {
	Entries []LootTableEntry
}

// LootTableEntry represents a single entry in a loot table
type LootTableEntry struct {
	ItemTemplateID string
	Weight         int
	MinCount       int
	MaxCount       int
}

// NewLootTable creates a new loot table
func NewLootTable(entries []LootTableEntry) *LootTable {
	return &LootTable{
		Entries: entries,
	}
}

// GenerateItems creates items based on the loot table
func (lt *LootTable) GenerateItems(world *ecs.World) []*ecs.Entity {
	var items []*ecs.Entity

	// Calculate total weight
	totalWeight := 0
	for _, entry := range lt.Entries {
		totalWeight += entry.Weight
	}

	// Generate items
	for _, entry := range lt.Entries {
		// Roll for this entry
		if rand.Intn(totalWeight) < entry.Weight {
			// Determine how many of this item to create
			count := entry.MinCount
			if entry.MaxCount > entry.MinCount {
				count += rand.Intn(entry.MaxCount - entry.MinCount + 1)
			}

			// Create the items
			for i := 0; i < count; i++ {
				// TODO: Create item from template
				// For now, we'll just create a placeholder
				item := world.CreateEntity()
				items = append(items, item)
			}
		}
	}

	return items
}
