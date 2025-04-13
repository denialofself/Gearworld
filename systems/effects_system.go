package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
	"strconv"
)

// Effect operation types
const (
	EffectAdd    = "add"
	EffectSet    = "set"
	EffectToggle = "toggle"
)

// EffectsSystem handles all types of effects (passive, active, temporary) in a unified way
type EffectsSystem struct {
	initialized bool
	world       *ecs.World
}

// NewEffectsSystem creates a new effects system
func NewEffectsSystem() *EffectsSystem {
	return &EffectsSystem{
		initialized: false,
	}
}

// Initialize sets up event listeners for the effects system
func (s *EffectsSystem) Initialize(world *ecs.World) {
	if s.initialized {
		return
	}

	s.world = world
	// Subscribe to rest events for passive healing
	world.GetEventManager().Subscribe(EventRest, func(event ecs.Event) {
		restEvent := event.(RestEvent)
		s.handleRest(world, restEvent)
	})

	// Subscribe to effects events
	world.GetEventManager().Subscribe(EventEffects, func(event ecs.Event) {
		effectEvent := event.(EffectsEvent)
		s.handleEffect(world, effectEvent)
	})

	s.initialized = true
}

// Update ensures the system is initialized
func (s *EffectsSystem) Update(world *ecs.World, dt float64) {
	// Ensure system is initialized with event handlers
	if !s.initialized {
		s.Initialize(world)
	}
}

// handleRest processes rest events and applies healing
func (s *EffectsSystem) handleRest(world *ecs.World, event RestEvent) {
	entityID := event.EntityID

	// Get stats component
	statsComp, hasStats := world.GetComponent(entityID, components.Stats)
	if !hasStats {
		return
	}

	stats := statsComp.(*components.StatsComponent)

	// Calculate healing amount based on healing factor
	healAmount := stats.HealingFactor

	// Store original health to calculate actual healing
	originalHealth := stats.Health

	// Use the reflection-based component system to apply the healing
	err := ApplyEntityEffect(
		world,
		entityID,
		"Stats",
		"Health",
		EffectAdd,
		healAmount,
	)

	if err != nil {
		GetMessageLog().Add(fmt.Sprintf("Error applying healing effect: %v", err))
		return
	}

	// Get current health after healing to determine how much was actually healed
	currentHealth, _ := components.GetComponentProperty(statsComp, "Health")
	currentMaxHealth, _ := components.GetComponentProperty(statsComp, "MaxHealth")

	// Calculate actual healing
	actualHealing := currentHealth.(int) - originalHealth

	// Show appropriate message based on healing result
	if actualHealing > 0 {
		GetMessageLog().Add("You rest for a moment and recover " + strconv.Itoa(actualHealing) + " health.")
	} else if currentHealth.(int) == currentMaxHealth.(int) {
		GetMessageLog().Add("You rest for a moment, but you're already at full health.")
	}
}

// handleEffect processes all types of effects
func (s *EffectsSystem) handleEffect(world *ecs.World, event EffectsEvent) {
	entityID := event.EntityID

	// Debug logging
	GetMessageLog().Add(fmt.Sprintf("DEBUG: Processing effect: %s on property %s with value %v",
		event.EffectType, event.Property, event.Value))

	switch event.EffectType {
	case EffectTypeHeal:
		s.applyHealingEffect(world, entityID, event)
	case EffectTypeDamage:
		s.applyDamageEffect(world, entityID, event)
	case EffectTypeStatBoost:
		s.applyStatBoostEffect(world, entityID, event)
	case EffectTypeFOVModify, EffectTypeLightSource:
		s.applyFOVEffect(world, entityID, event)
	}
}

// applyHealingEffect handles healing effects
func (s *EffectsSystem) applyHealingEffect(world *ecs.World, entityID ecs.EntityID, event EffectsEvent) {
	// Get stats component
	statsComp, hasStats := world.GetComponent(entityID, components.Stats)
	if !hasStats {
		return
	}

	stats := statsComp.(*components.StatsComponent)

	// Get healing amount
	var healAmount int
	switch v := event.Value.(type) {
	case int:
		healAmount = v
	case float64:
		healAmount = int(v)
	default:
		return // Unsupported value type
	}

	// Apply healing (don't exceed max health)
	oldHealth := stats.Health
	stats.Health += healAmount
	if stats.Health > stats.MaxHealth {
		stats.Health = stats.MaxHealth
	}

	// Calculate actual healing done
	actualHealing := stats.Health - oldHealth

	// Handle message display
	if actualHealing > 0 {
		message := event.DisplayText
		if message == "" {
			message = "You feel better"
		}

		message += " and recover " + strconv.Itoa(actualHealing) + " health."
		GetMessageLog().Add(message)
	} else if stats.Health == stats.MaxHealth && event.Source == "rest" {
		GetMessageLog().Add(event.DisplayText + ", but you're already at full health.")
	} else if actualHealing == 0 && event.Source != "rest" {
		GetMessageLog().Add("You're already at full health.")
	}
}

// applyDamageEffect handles damage effects
func (s *EffectsSystem) applyDamageEffect(world *ecs.World, entityID ecs.EntityID, event EffectsEvent) {
	// Get stats component
	statsComp, hasStats := world.GetComponent(entityID, components.Stats)
	if !hasStats {
		return
	}

	stats := statsComp.(*components.StatsComponent)

	// Get damage amount
	var damageAmount int
	switch v := event.Value.(type) {
	case int:
		damageAmount = v
	case float64:
		damageAmount = int(v)
	default:
		return // Unsupported value type
	}

	// Apply damage
	stats.Health -= damageAmount

	// Handle message display
	if event.DisplayText != "" {
		GetMessageLog().Add(event.DisplayText)
	} else {
		GetMessageLog().Add("You take " + strconv.Itoa(damageAmount) + " damage.")
	}

	// Check for death
	if stats.Health <= 0 {
		stats.Health = 0
		// We could emit a death event here
		GetMessageLog().Add("You have died!")
	}
}

// applyStatBoostEffect handles stat boost effects
func (s *EffectsSystem) applyStatBoostEffect(world *ecs.World, entityID ecs.EntityID, event EffectsEvent) {
	// Get stats component
	statsComp, hasStats := world.GetComponent(entityID, components.Stats)
	if !hasStats {
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Cannot apply stat effect - entity has no Stats component"))
		return
	}

	stats := statsComp.(*components.StatsComponent)

	// Get boost value
	var boostValue int
	switch v := event.Value.(type) {
	case int:
		boostValue = v
	case float64:
		boostValue = int(v)
	default:
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Unsupported value type for stat boost: %T", event.Value))
		return // Unsupported value type
	}

	// Debug logging - before stats
	GetMessageLog().Add(fmt.Sprintf("DEBUG: Before stat change - %s: %d",
		event.Property, getStatValue(stats, event.Property)))

	// Apply boost based on property
	switch event.Property {
	case "Attack":
		stats.Attack += boostValue
	case "Defense":
		stats.Defense += boostValue
	case "MaxHealth":
		stats.MaxHealth += boostValue
		// If increasing max health, also increase current health
		if boostValue > 0 {
			stats.Health += boostValue
		}
	}

	// Debug logging - after stats
	GetMessageLog().Add(fmt.Sprintf("DEBUG: After stat change - %s: %d",
		event.Property, getStatValue(stats, event.Property)))

	// Handle message display
	if event.DisplayText != "" {
		GetMessageLog().Add(event.DisplayText)
	} else {
		changeVerb := "decreases"
		if boostValue > 0 {
			changeVerb = "increases"
		}
		GetMessageLog().Add("Your " + event.Property + " " +
			changeVerb + " by " +
			strconv.Itoa(abs(boostValue)) + ".")
	}
}

// Helper function to get stat value by property name
func getStatValue(stats *components.StatsComponent, property string) int {
	switch property {
	case "Attack":
		return stats.Attack
	case "Defense":
		return stats.Defense
	case "MaxHealth":
		return stats.MaxHealth
	case "Health":
		return stats.Health
	default:
		return 0
	}
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// applyFOVEffect handles FOV-related effects
func (s *EffectsSystem) applyFOVEffect(world *ecs.World, entityID ecs.EntityID, event EffectsEvent) {
	// Get FOV component
	fovComp, hasFOV := world.GetComponent(entityID, components.FOV)
	if !hasFOV {
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Cannot apply FOV effect - entity has no FOV component"))
		return
	}

	fov := fovComp.(*components.FOVComponent)

	// Debug logging - before FOV changes
	GetMessageLog().Add(fmt.Sprintf("DEBUG: Before FOV change - Range: %d, LightSource: %v, LightRange: %d",
		fov.Range, fov.LightSource, fov.LightRange))

	// Get effect value
	var effectValue int
	switch v := event.Value.(type) {
	case int:
		effectValue = v
	case float64:
		effectValue = int(v)
	case bool:
		if event.Property == "LightSource" {
			fov.LightSource = v
			GetMessageLog().Add(fmt.Sprintf("DEBUG: Set LightSource to %v", v))
		}
		return
	default:
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Unsupported value type for FOV effect: %T", event.Value))
		return // Unsupported value type
	}

	// Apply effect based on property
	switch event.Property {
	case "Range":
		fov.Range += effectValue
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Added %d to FOV Range, now %d", effectValue, fov.Range))
	case "LightRange":
		fov.LightRange += effectValue
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Added %d to LightRange, now %d", effectValue, fov.LightRange))
		if effectValue > 0 {
			fov.LightSource = true
			GetMessageLog().Add("DEBUG: Set LightSource to true due to positive LightRange")
		} else if fov.LightRange <= 0 {
			fov.LightSource = false
			GetMessageLog().Add("DEBUG: Set LightSource to false due to non-positive LightRange")
		}
	}

	// Debug logging - after FOV changes
	GetMessageLog().Add(fmt.Sprintf("DEBUG: After FOV change - Range: %d, LightSource: %v, LightRange: %d",
		fov.Range, fov.LightSource, fov.LightRange))

	// Handle message display
	if event.DisplayText != "" {
		GetMessageLog().Add(event.DisplayText)
	} else if event.Property == "LightRange" && effectValue > 0 {
		GetMessageLog().Add("The area around you is illuminated.")
	} else if event.Property == "Range" && effectValue > 0 {
		GetMessageLog().Add("Your vision range has increased.")
	}
}

// CreateItemEffect creates a new ItemEffect from parameters
func CreateItemEffect(componentName, propertyName, operation string, value interface{}) components.ItemEffect {
	return components.ItemEffect{
		Component: componentName,
		Property:  propertyName,
		Operation: operation,
		Value:     value,
	}
}

// ApplyEntityEffects applies multiple effects to an entity at once
func ApplyEntityEffects(world *ecs.World, entityID ecs.EntityID, effects []components.ItemEffect) error {
	if len(effects) == 0 {
		return nil // No effects to apply
	}

	for _, effect := range effects {
		err := ApplyEntityEffect(world, entityID, effect.Component, effect.Property, effect.Operation, effect.Value)
		if err != nil {
			return fmt.Errorf("failed to apply effect to %s.%s: %v", effect.Component, effect.Property, err)
		}
	}

	return nil
}

// RemoveEntityEffects removes (reverses) multiple effects from an entity
func RemoveEntityEffects(world *ecs.World, entityID ecs.EntityID, effects []components.ItemEffect) error {
	if len(effects) == 0 {
		return nil // No effects to remove
	}

	for _, effect := range effects {
		inverseOp, inverseVal, err := CreateInverseEffect(effect.Component, effect.Property, effect.Operation, effect.Value)
		if err != nil {
			return fmt.Errorf("failed to create inverse effect for %s.%s: %v", effect.Component, effect.Property, err)
		}

		err = ApplyEntityEffect(world, entityID, effect.Component, effect.Property, inverseOp, inverseVal)
		if err != nil {
			return fmt.Errorf("failed to apply inverse effect to %s.%s: %v", effect.Component, effect.Property, err)
		}
	}

	return nil
}

// ApplyEntityEffect applies an effect to an entity's component
func ApplyEntityEffect(world *ecs.World, entityID ecs.EntityID, componentName string,
	propertyName string, operation string, value interface{}) error {

	// Get component ID
	compID, exists := components.GetComponentIDByName(componentName)
	if !exists {
		return fmt.Errorf("unknown component type: %s", componentName)
	}

	// Get the component
	comp, hasComp := world.GetComponent(entityID, compID)
	if !hasComp {
		return fmt.Errorf("entity %d lacks %s component", entityID, componentName)
	}

	// Apply the effect to the component
	return ApplyComponentEffect(comp, propertyName, operation, value)
}

// ApplyComponentEffect applies an effect to a component's property
func ApplyComponentEffect(comp interface{}, propertyName string, operation string, value interface{}) error {
	// Get current value
	currentValue, err := components.GetComponentProperty(comp, propertyName)
	if err != nil {
		return fmt.Errorf("error getting property: %v", err)
	}

	var newValue interface{}

	// Calculate new value based on operation
	switch operation {
	case EffectAdd:
		// Addition operation - supported for numeric types
		switch current := currentValue.(type) {
		case int:
			switch v := value.(type) {
			case int:
				newValue = current + v
			case float64:
				newValue = current + int(v)
			default:
				return fmt.Errorf("cannot add %T to int", value)
			}
		case int64:
			switch v := value.(type) {
			case int:
				newValue = current + int64(v)
			case float64:
				newValue = current + int64(v)
			case int64:
				newValue = current + v
			default:
				return fmt.Errorf("cannot add %T to int64", value)
			}
		case float64:
			switch v := value.(type) {
			case int:
				newValue = current + float64(v)
			case float64:
				newValue = current + v
			default:
				return fmt.Errorf("cannot add %T to float64", value)
			}
		default:
			return fmt.Errorf("addition not supported for %T", currentValue)
		}

	case EffectSet:
		// Set operation - just use the new value
		newValue = value

	case EffectToggle:
		// Toggle operation - only for booleans
		if current, ok := currentValue.(bool); ok {
			newValue = !current
		} else {
			return fmt.Errorf("toggle operation only supports boolean values, got %T", currentValue)
		}

	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	// Set the new value
	err = components.SetComponentProperty(comp, propertyName, newValue)
	if err != nil {
		return fmt.Errorf("error setting property: %v", err)
	}

	// Special cases for component interactions
	HandleSpecialCases(comp, propertyName)

	return nil
}

// HandleSpecialCases handles special interactions between component properties
func HandleSpecialCases(comp interface{}, propertyName string) {
	// Check StatsComponent - cap Health at MaxHealth
	if statsComp, ok := comp.(*components.StatsComponent); ok && propertyName == "Health" {
		if statsComp.Health > statsComp.MaxHealth {
			statsComp.Health = statsComp.MaxHealth
		}
	}

	// Check FOVComponent - light source based on light range
	if fovComp, ok := comp.(*components.FOVComponent); ok && propertyName == "LightRange" {
		fovComp.LightSource = fovComp.LightRange > 0
	}
}

// GetDefaultValueForProperty returns the default value for a given component and property
func GetDefaultValueForProperty(componentName, propertyName string) (interface{}, error) {
	switch componentName {
	case "Stats":
		switch propertyName {
		case "Health", "MaxHealth":
			return 10, nil // Default health and max health
		case "Attack":
			return 1, nil // Default attack
		case "Defense":
			return 0, nil // Default defense
		}
	case "FOV":
		switch propertyName {
		case "Range":
			return 8, nil // Default FOV range
		case "LightRange":
			return 0, nil // Default light range
		case "LightSource":
			return false, nil // Default light source state
		}
	}

	return nil, fmt.Errorf("no default value defined for %s.%s", componentName, propertyName)
}

// CreateInverseEffect creates the inverse of an effect for removal
func CreateInverseEffect(componentName, propertyName, operation string, value interface{}) (string, interface{}, error) {
	switch operation {
	case EffectAdd:
		// For numeric values, negate them
		switch v := value.(type) {
		case int:
			return EffectAdd, -v, nil
		case float64:
			return EffectAdd, -v, nil
		case int64:
			return EffectAdd, -v, nil
		default:
			return "", nil, fmt.Errorf("cannot create inverse for type %T with add operation", value)
		}

	case EffectSet:
		// For "set" operations, use defaults based on property
		defaultValue, err := GetDefaultValueForProperty(componentName, propertyName)
		if err != nil {
			return "", nil, err
		}
		return EffectSet, defaultValue, nil

	case EffectToggle:
		// For toggle operations, just toggle again
		return EffectToggle, nil, nil

	default:
		return "", nil, fmt.Errorf("unknown operation: %s", operation)
	}
}
