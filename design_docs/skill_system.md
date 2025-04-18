# Skill System Design

## Overview

The skill system is designed to be flexible and extensible, supporting both passive and active skills that can affect various game systems, particularly combat. The system uses an event-based approach to maintain proper ECS principles while allowing skills to interact with different game systems.

## Core Components

### Skill Component

```go
type Skill struct {
    Name        string
    Level       int
    Experience  int
    Category    string
    // Combat-specific fields
    IsPassive   bool
    CombatTags  []string  // e.g., "melee", "ranged", "defensive", "offensive"
    // ... other fields
}

type SkillComponent struct {
    Skills map[string]Skill
}
```

## Combat Integration

### Passive Skills

Passive skills are automatically applied during combat based on their tags. The combat system reads the skill component and applies relevant effects.

```go
func (s *CombatSystem) ProcessCombat(world *ecs.World, event CombatEvent) {
    // Get attacker's skills
    if skillComp, ok := world.GetComponent(event.AttackerID, components.Skill); ok {
        // Apply passive combat skill effects
        for _, skill := range skillComp.Skills {
            if skill.IsPassive && contains(skill.CombatTags, "offensive") {
                s.applyPassiveSkillEffect(event, skill)
            }
        }
    }

    // Get defender's skills
    if skillComp, ok := world.GetComponent(event.DefenderID, components.Skill); ok {
        // Apply passive combat skill effects
        for _, skill := range skillComp.Skills {
            if skill.IsPassive && contains(skill.CombatTags, "defensive") {
                s.applyPassiveSkillEffect(event, skill)
            }
        }
    }

    // Process the actual combat
    // ...
}

func (s *CombatSystem) applyPassiveSkillEffect(event CombatEvent, skill Skill) {
    switch skill.Name {
    case "dodge":
        // Modify dodge chance
        event.DodgeChance += skill.Level * 0.05
    case "melee_combat":
        // Modify hit chance
        event.HitChance += skill.Level * 0.03
    // ... other passive effects
    }
}
```

### Active Skills

Active skills generate their own combat events when used. The skill system handles the activation and event generation.

```go
func (s *SkillSystem) HandleSkillUse(world *ecs.World, event SkillUseEvent) {
    skill := event.Skill
    
    switch skill.Name {
    case "rapid_shot":
        // Generate multiple attack events
        for i := 0; i < 3; i++ {
            world.EmitEvent(CombatEvent{
                AttackerID: event.EntityID,
                TargetID:   event.TargetID,
                AttackType: "ranged",
                // ... other combat parameters
            })
        }
    case "power_attack":
        // Generate a single powerful attack
        world.EmitEvent(CombatEvent{
            AttackerID: event.EntityID,
            TargetID:   event.TargetID,
            AttackType: "melee",
            DamageMultiplier: 2.0,
            // ... other combat parameters
        })
    // ... other active skills
    }
}
```

## Example Skill Definitions

```go
// Passive skills
dodge := Skill{
    Name: "dodge",
    IsPassive: true,
    CombatTags: []string{"defensive"},
    // ... other fields
}

meleeCombat := Skill{
    Name: "melee_combat",
    IsPassive: true,
    CombatTags: []string{"offensive", "melee"},
    // ... other fields
}

// Active skills
rapidShot := Skill{
    Name: "rapid_shot",
    IsPassive: false,
    CombatTags: []string{"offensive", "ranged"},
    // ... other fields
}

powerAttack := Skill{
    Name: "power_attack",
    IsPassive: false,
    CombatTags: []string{"offensive", "melee"},
    // ... other fields
}
```

## Design Benefits

1. **Clear Separation**: Distinction between passive and active skills
2. **ECS Compliance**: Systems only read components and emit events
3. **Flexibility**: Easy to add new passive effects and active skills
4. **Tag-based Organization**: Skills can be categorized for different contexts
5. **Maintainability**: Each system handles its specific responsibilities

## System Responsibilities

### Combat System
- Reads relevant skills from components
- Applies passive effects to combat calculations
- Processes the actual combat

### Skill System
- Activates skills
- Generates appropriate events
- Manages skill progression

## Future Extensions

The system can be extended to support:
1. Non-combat skills (crafting, survival, etc.)
2. Skill synergies and combinations
3. Skill trees and prerequisites
4. Temporary skill effects and buffs
5. Skill-based equipment requirements 