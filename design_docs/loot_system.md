# Loot System Design

## Overview

The loot system is designed to handle both static and random loot generation for entities that can contain items (monsters, containers, etc.). It uses the existing item tagging system and provides a flexible way to define loot tables and drop rules.

## Core Components

### Loot Component

```go
type LootComponent struct {
    // Static items that are always present
    StaticItems []ecs.EntityID
    
    // Loot tables for random drops
    LootTables []LootTable
    
    // Tags that affect loot generation
    Tags []string
}

type LootTable struct {
    Name        string
    Rolls       int           // Number of times to roll on this table
    Items       []LootEntry   // Possible items to drop
    Conditions  []string      // Tags that must be present to use this table
}

type LootEntry struct {
    ItemID      string        // Reference to item template
    Weight      int           // Relative chance of being selected
    MinCount    int           // Minimum number to drop
    MaxCount    int           // Maximum number to drop
    Tags        []string      // Tags that must match for this entry to be valid
}
```

## Loot Generation

### Loot System

```go
type LootSystem struct {
    // Handles loot generation and distribution
}

func (s *LootSystem) GenerateLoot(world *ecs.World, entityID ecs.EntityID) {
    // Get the loot component
    lootComp, ok := world.GetComponent(entityID, components.Loot)
    if !ok {
        return
    }

    // Create a new container entity for the loot
    containerID := s.createLootContainer(world, entityID)

    // Add static items
    for _, itemID := range lootComp.StaticItems {
        s.addItemToContainer(world, containerID, itemID)
    }

    // Process loot tables
    for _, table := range lootComp.LootTables {
        if s.tableConditionsMet(table, lootComp.Tags) {
            s.processLootTable(world, containerID, table)
        }
    }
}

func (s *LootSystem) processLootTable(world *ecs.World, containerID ecs.EntityID, table LootTable) {
    for i := 0; i < table.Rolls; i++ {
        entry := s.selectLootEntry(table.Items)
        if entry != nil {
            count := rand.Intn(entry.MaxCount-entry.MinCount+1) + entry.MinCount
            for j := 0; j < count; j++ {
                s.createItemFromTemplate(world, containerID, entry.ItemID)
            }
        }
    }
}
```

## Integration with Existing Systems

### Monster Death

```go
func (s *CombatSystem) handleEntityDeath(world *ecs.World, entityID ecs.EntityID) {
    // ... existing death handling ...

    // Generate loot
    world.EmitEvent(LootGenerationEvent{
        EntityID: entityID,
    })
}
```

### Container Interaction

```go
func (s *InteractionSystem) handleContainerOpen(world *ecs.World, containerID ecs.EntityID) {
    // Generate loot if not already generated
    if !s.isLootGenerated(containerID) {
        world.EmitEvent(LootGenerationEvent{
            EntityID: containerID,
        })
    }
}
```

## Example Loot Definitions

```json
{
    "loot_tables": {
        "goblin_basic": {
            "rolls": 2,
            "items": [
                {
                    "item_id": "rusty_dagger",
                    "weight": 30,
                    "min_count": 1,
                    "max_count": 1,
                    "tags": ["weapon", "melee"]
                },
                {
                    "item_id": "copper_coin",
                    "weight": 70,
                    "min_count": 1,
                    "max_count": 5,
                    "tags": ["currency"]
                }
            ]
        },
        "goblin_rare": {
            "rolls": 1,
            "conditions": ["elite"],
            "items": [
                {
                    "item_id": "goblin_charm",
                    "weight": 100,
                    "min_count": 1,
                    "max_count": 1,
                    "tags": ["magic", "charm"]
                }
            ]
        }
    }
}
```

## Monster Template Integration

```json
{
    "name": "goblin",
    "components": {
        "loot": {
            "static_items": ["goblin_ear"],
            "loot_tables": ["goblin_basic"],
            "tags": ["humanoid", "goblin"]
        }
    }
}
```

## Container Template Integration

```json
{
    "name": "wooden_chest",
    "components": {
        "loot": {
            "static_items": ["rusty_key"],
            "loot_tables": ["dungeon_common"],
            "tags": ["container", "wooden"]
        }
    }
}
```

## Design Benefits

1. **Flexibility**: Supports both static and random loot
2. **Reusability**: Loot tables can be shared between different entities
3. **Conditional Drops**: Tags can control which loot tables are used
4. **Weighted Randomness**: Fine control over drop chances
5. **Integration**: Works with existing item and tagging systems

## System Responsibilities

### Loot System
- Generates loot for entities
- Processes loot tables
- Creates items from templates
- Manages container creation

### Combat System
- Triggers loot generation on death
- Applies death-related tags

### Interaction System
- Triggers loot generation for containers
- Manages container state

## Future Extensions

The system can be extended to support:
1. Dynamic loot tables based on game state
2. Quest-specific loot drops
3. Player luck/skill affecting drops
4. Loot quality tiers
5. Custom loot generation rules 