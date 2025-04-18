# Monster Abilities Design

## Overview

The monster abilities system allows monsters to use special abilities during combat. These abilities can have immediate effects (like damage) and/or ongoing effects (like damage over time). The system is designed to be flexible and extensible, allowing for a wide variety of ability types and effects.

## Core Components

### MonsterAbilityComponent

```go
type MonsterAbilityComponent struct {
    Abilities []MonsterAbility
}

type MonsterAbility struct {
    Name        string
    Description string
    Type        string    // "active" or "passive"
    Cooldown    int       // Number of turns before ability can be used again
    CurrentCD   int       // Current cooldown counter
    Range       int       // Range of the ability (0 for self, 1 for adjacent, etc.)
    Cost        int       // Action point cost to use the ability
    Effects     []EffectDefinition // Effects to apply when ability is used
    Trigger     string    // When to use the ability ("on_attack", "on_hit", "on_turn_start", etc.)
}
```

### EffectComponent

```go
type EffectComponent struct {
    Type      string        // Type of effect (e.g., "damage_over_time")
    Duration  int           // Total duration in turns
    Remaining int           // Remaining duration
    Damage    int           // Damage per interval (for damage effects)
    Interval  int           // Turns between effect applications
    Source    ecs.EntityID  // Entity that caused the effect
}
```

## System Responsibilities

### Combat System
- Determines if an ability can be used (range, cooldown, action points)
- Handles ability hit/miss calculations
- Applies immediate effects
- Emits events for ongoing effects
- Manages ability cooldowns
- Consumes action points for ability use

### Effects System
- Tracks duration of effects
- Applies periodic effects (e.g., damage over time)
- Manages effect stacks
- Removes effects when they expire
- Handles effect cleanup

## Ability Types

### Active Abilities
- Require explicit activation by the monster
- Have cooldowns and action point costs
- Examples:
  - "Bleeding Bite" - Causes damage over time
  - "Poison Spray" - Applies poison in an area
  - "Healing Surge" - Restores health

### Passive Abilities
- Trigger automatically based on conditions
- May or may not have cooldowns
- Examples:
  - "Thick Hide" - Reduces incoming damage
  - "Regeneration" - Heals over time
  - "Retaliation" - Counterattacks when hit

## Effect Types

### Immediate Effects
- Applied once when ability hits
- Examples:
  - Direct damage
  - Healing
  - Stat modifications

### Ongoing Effects
- Applied over multiple turns
- Examples:
  - Damage over time
  - Healing over time
  - Stat modifications over time

## Example Ability Definition

```json
{
    "name": "wire_spider",
    "components": {
        "monster_ability": {
            "abilities": [
                {
                    "name": "Bleeding Bite",
                    "description": "Inflicts a bleeding wound that deals 1d4 damage per turn",
                    "type": "active",
                    "cooldown": 3,
                    "current_cd": 0,
                    "range": 1,
                    "cost": 2,
                    "trigger": "on_attack",
                    "effects": [
                        {
                            "type": "damage_over_time",
                            "duration": "1d4",
                            "damage": 1,
                            "interval": 1
                        }
                    ]
                }
            ]
        }
    }
}
```

## Combat Flow

1. Monster's turn begins
2. AI system checks for available abilities
3. If ability can be used:
   - Combat system checks range and action points
   - Rolls to hit
   - If hit:
     - Applies immediate effects
     - Emits effect event
     - Starts cooldown
     - Consumes action points
4. Effects system:
   - Creates effect components
   - Tracks duration
   - Applies periodic effects
   - Removes expired effects

## Design Benefits

1. **Separation of Concerns**
   - Combat system handles tactical decisions
   - Effects system manages ongoing state
   - Each system can be modified independently

2. **Flexibility**
   - Easy to add new ability types
   - Easy to add new effect types
   - Supports both active and passive abilities

3. **Extensibility**
   - New triggers can be added
   - New effect types can be added
   - Ability templates can be modified without code changes

4. **Maintainability**
   - Clear system boundaries
   - Well-defined component structure
   - Modular design

## Future Extensions

1. **Area of Effect Abilities**
   - Add support for abilities that affect multiple targets
   - Define area shapes (circle, cone, line)

2. **Conditional Effects**
   - Effects that trigger based on game state
   - Effects that modify other effects

3. **Ability Combinations**
   - Effects that interact with other effects
   - Combo systems

4. **Visual Effects**
   - Particle systems for ability animations
   - Sound effects for abilities

5. **Ability Learning**
   - Monsters that learn new abilities
   - Ability progression systems 