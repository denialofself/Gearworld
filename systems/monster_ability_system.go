package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
)

// TurnEvent represents the start or end of a turn
type TurnEvent struct {
	EntityID ecs.EntityID
}

func (e TurnEvent) Type() ecs.EventType {
	return "turn"
}

// MonsterAbilitySystem handles monster abilities and their effects
type MonsterAbilitySystem struct {
	world         *ecs.World
	initialized   bool
	effectsSystem *EffectsSystem
}

// NewMonsterAbilitySystem creates a new monster ability system
func NewMonsterAbilitySystem() *MonsterAbilitySystem {
	return &MonsterAbilitySystem{
		effectsSystem: NewEffectsSystem(),
	}
}

// Initialize sets up the system with the world and registers event listeners
func (s *MonsterAbilitySystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	s.world = world
	s.effectsSystem.Initialize(world)

	// Subscribe to combat attack events
	world.GetEventManager().Subscribe(EventCombatAttack, func(event ecs.Event) {
		if attackEvent, ok := event.(CombatAttackEvent); ok {
			s.handleAttack(world, attackEvent)
		}
	})

	// Subscribe to turn events
	world.GetEventManager().Subscribe("turn", func(event ecs.Event) {
		if turnEvent, ok := event.(TurnEvent); ok {
			s.handleTurnEnd(turnEvent)
		}
	})

	s.initialized = true
}

// handleAttack processes abilities triggered by an attack
func (s *MonsterAbilitySystem) handleAttack(world *ecs.World, event CombatAttackEvent) {
	attackerName := getEntityName(world, event.AttackerID)
	defenderName := getEntityName(world, event.DefenderID)
	GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Received combat attack event - %s attacking %s", attackerName, defenderName))

	// Get the attacker's monster ability component
	if abilityComp, exists := world.GetComponent(event.AttackerID, components.MonsterAbility); exists {
		if abilities, ok := abilityComp.(*components.MonsterAbilityComponent); ok {
			GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: %s has %d abilities", attackerName, len(abilities.Abilities)))

			// Get attacker's stats
			if statsComp, exists := world.GetComponent(event.AttackerID, components.Stats); exists {
				if stats, ok := statsComp.(*components.StatsComponent); ok {
					GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: %s has %d action points", attackerName, stats.ActionPoints))

					// Check each ability
					for _, ability := range abilities.Abilities {
						GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Checking ability '%s' (trigger: %s, cooldown: %d/%d, cost: %d)",
							ability.Name, ability.Trigger, ability.CurrentCD, ability.Cooldown, ability.Cost))

						// Skip if not an attack trigger or on cooldown
						if ability.Trigger != components.TriggerOnAttack {
							GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Skipping '%s' - wrong trigger type", ability.Name))
							continue
						}
						if ability.CurrentCD > 0 {
							GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Skipping '%s' - on cooldown", ability.Name))
							continue
						}

						// Check if we have enough action points
						if stats.ActionPoints < ability.Cost {
							GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Skipping '%s' - not enough action points (%d < %d)",
								ability.Name, stats.ActionPoints, ability.Cost))
							continue
						}

						GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Triggering '%s' ability", ability.Name))

						// Apply the ability's effects to the defender
						for _, effect := range ability.Effects {
							// Create and apply the effect
							gameEffect := s.effectsSystem.CreateGameEffect(
								effect.Type,
								effect.Operation,
								effect.Value,
								effect.Duration,
								event.AttackerID,
								effect.Target.Component,
								effect.Target.Property,
							)
							s.effectsSystem.ApplyEntityEffects(world, event.DefenderID, []components.GameEffect{gameEffect})

							// Log the ability use
							GetMessageLog().AddCombat(fmt.Sprintf("%s's %s causes %s to start bleeding!", attackerName, ability.Name, defenderName))
							GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Applied effect - type: %s, operation: %s, value: %v, duration: %d",
								effect.Type, effect.Operation, effect.Value, effect.Duration))
						}

						// Deduct action points
						stats.ActionPoints -= ability.Cost
						GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Deducted %d action points from %s", ability.Cost, attackerName))

						// Set cooldown
						ability.CurrentCD = ability.Cooldown
						GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: Set '%s' cooldown to %d", ability.Name, ability.Cooldown))
					}
				}
			}
		}
	} else {
		GetDebugLog().Add(fmt.Sprintf("MonsterAbilitySystem: %s has no abilities", attackerName))
	}
}

// handleTurnStart processes abilities triggered at the start of a turn
func (s *MonsterAbilitySystem) handleTurnStart(event ecs.Event) {
	turnEvent, ok := event.(TurnEvent)
	if !ok {
		return
	}

	// Get the entity's monster ability component
	abilityComp, ok := s.world.GetComponent(turnEvent.EntityID, components.MonsterAbility)
	if !ok {
		return
	}
	abilityComponent := abilityComp.(*components.MonsterAbilityComponent)

	// Update cooldowns
	abilityComponent.UpdateCooldowns()

	// Check each ability for on_turn_start trigger
	for i := range abilityComponent.Abilities {
		ability := &abilityComponent.Abilities[i]
		if ability.Trigger == components.TriggerOnTurnStart && ability.CurrentCD == 0 {
			// Apply the ability's effects to the entity itself
			for _, effect := range ability.Effects {
				// Set the effect's source to the entity
				effect.Source = turnEvent.EntityID
				// Add the effect to the entity
				if effectComp, ok := s.world.GetComponent(turnEvent.EntityID, components.Effect); ok {
					effectComponent := effectComp.(*components.EffectComponent)
					effectComponent.AddEffect(effect)
				}
			}
			// Start the cooldown
			ability.CurrentCD = ability.Cooldown
		}
	}
}

// handleTurnEnd processes abilities triggered at the end of a turn
func (s *MonsterAbilitySystem) handleTurnEnd(event ecs.Event) {
	turnEvent, ok := event.(TurnEvent)
	if !ok {
		return
	}

	// Get the entity's monster ability component
	abilityComp, ok := s.world.GetComponent(turnEvent.EntityID, components.MonsterAbility)
	if !ok {
		return
	}
	abilityComponent := abilityComp.(*components.MonsterAbilityComponent)

	// Check each ability for on_turn_end trigger
	for i := range abilityComponent.Abilities {
		ability := &abilityComponent.Abilities[i]
		if ability.Trigger == components.TriggerOnTurnEnd && ability.CurrentCD == 0 {
			// Apply the ability's effects to the entity itself
			for _, effect := range ability.Effects {
				// Set the effect's source to the entity
				effect.Source = turnEvent.EntityID
				// Add the effect to the entity
				if effectComp, ok := s.world.GetComponent(turnEvent.EntityID, components.Effect); ok {
					effectComponent := effectComp.(*components.EffectComponent)
					effectComponent.AddEffect(effect)
				}
			}
			// Start the cooldown
			ability.CurrentCD = ability.Cooldown
		}
	}
}

// Update processes the system's logic
func (s *MonsterAbilitySystem) Update(world *ecs.World, dt float64) {
	// No-op for now
}
