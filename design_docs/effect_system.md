# Unified Effect System Design

## Overview

This document outlines the design for a unified effect system that will handle all types of effects in the game, including item effects, monster abilities, and environmental effects. The system builds upon the existing `ItemEffect` implementation while adding support for duration-based effects, damage over time, and more complex effect interactions.

## Core Components

### Effect Component

```go
type Effect struct {
    Type        string      // Type of effect (heal, damage, stat_boost, etc.)
    Component   string      // Component to affect
    Property    string      // Property to modify
    Operation   string      // Operation to perform
    Value       interface{} // Value to apply
    Duration    int         // Duration in turns (0 for instant)
    Interval    int         // Interval between ticks (for DoT effects)
    Source      string      // Source of the effect (item_id, ability_id, etc.)
    Tags        []string    // Additional tags for effect categorization
}
```

### Effect Types

```go
const (
    EffectTypeInstant     = "instant"     // Immediate effect
    EffectTypeDuration    = "duration"    // Effect with duration
    EffectTypePeriodic    = "periodic"    // Effect that ticks at intervals
    EffectTypeConditional = "conditional" // Effect that applies under conditions
)
```

### Effect Operations

```go
const (
    EffectOpAdd      = "add"      // Add to current value
    EffectOpSubtract = "subtract" // Subtract from current value
    EffectOpMultiply = "multiply" // Multiply current value
    EffectOpSet      = "set"      // Set to new value
    EffectOpToggle   = "toggle"   // Toggle boolean value
)
```

## System Architecture

### EffectSystem

```go
type EffectSystem struct {
    world *ecs.World
    activeEffects map[ecs.EntityID][]*Effect
}

func (s *EffectSystem) ApplyEffect(entityID ecs.EntityID, effect *Effect) error
func (s *EffectSystem) RemoveEffect(entityID ecs.EntityID, effect *Effect) error
func (s *EffectSystem) Update(dt float64)
func (s *EffectSystem) GetActiveEffects(entityID ecs.EntityID) []*Effect
```

### Effect Events

```go
type EffectEvent struct {
    EntityID    ecs.EntityID
    Effect      *Effect
    EventType   string // "apply", "remove", "tick"
    Success     bool
    Message     string
}
```

## Implementation Details

### Effect Application

1. **Instant Effects**
   - Applied immediately
   - No duration or interval
   - Example: Healing potion

2. **Duration Effects**
   - Applied immediately
   - Removed after duration expires
   - Example: Temporary stat boost

3. **Periodic Effects**
   - Applied at regular intervals
   - Can have duration
   - Example: Poison damage over time

4. **Conditional Effects**
   - Applied when conditions are met
   - Can be instant or duration-based
   - Example: Bonus damage when health is low

### Effect Stacking

- Multiple effects of the same type can stack
- Stacking behavior defined by effect type
- Maximum stacks can be defined per effect

### Effect Removal

- Effects can be removed by:
  - Duration expiration
  - Manual removal
  - Condition change
  - Entity death

## Integration Points

### Item System Integration
- Items can apply effects when:
  - Equipped
  - Used
  - Consumed

### Monster Ability Integration
- Abilities can apply effects when:
  - Activated
  - Hit target
  - Miss target

### Environment Integration
- Environment can apply effects when:
  - Entity enters area
  - Entity stays in area
  - Entity leaves area

## Example Usage

### Healing Potion
```go
effect := &Effect{
    Type:      EffectTypeInstant,
    Component: "Stats",
    Property:  "Health",
    Operation: EffectOpAdd,
    Value:     20,
    Source:    "healing_potion",
    Tags:      []string{"healing", "consumable"},
}
```

### Poison Effect
```go
effect := &Effect{
    Type:      EffectTypePeriodic,
    Component: "Stats",
    Property:  "Health",
    Operation: EffectOpSubtract,
    Value:     5,
    Duration:  5,
    Interval:  1,
    Source:    "poison_ability",
    Tags:      []string{"damage", "dot", "poison"},
}
```

### Temporary Stat Boost
```go
effect := &Effect{
    Type:      EffectTypeDuration,
    Component: "Stats",
    Property:  "Strength",
    Operation: EffectOpAdd,
    Value:     5,
    Duration:  3,
    Source:    "strength_potion",
    Tags:      []string{"buff", "stat_boost"},
}
```

## Migration Strategy

1. Phase 1: Core Implementation
   - Create new Effect component
   - Implement EffectSystem
   - Add basic effect types

2. Phase 2: Item System Integration
   - Update ItemEffect to use new system
   - Migrate existing item effects
   - Update item templates

3. Phase 3: Monster Ability Integration
   - Add ability effect support
   - Update monster templates
   - Implement ability effects

4. Phase 4: Environment Integration
   - Add environmental effects
   - Update map generation
   - Implement area effects

## Benefits

1. Unified Effect Handling
   - Single system for all effects
   - Consistent effect application
   - Easier effect management

2. Extensibility
   - Easy to add new effect types
   - Flexible effect configuration
   - Support for complex interactions

3. Maintainability
   - Centralized effect logic
   - Clear effect definitions
   - Easy to debug and modify

4. Performance
   - Efficient effect processing
   - Optimized effect updates
   - Minimal memory overhead 