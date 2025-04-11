package ecs

// World manages all entities and components
type World struct {
	entities map[EntityID]*Entity
	// Store components as map[EntityID]map[ComponentID]Component
	components map[EntityID]ComponentMap
	// Systems slice to store all systems
	systems []System
	// Tag-based entity lookup for quick access
	entityTags map[string]map[EntityID]bool
	// Event manager for system communication
	eventManager *EventManager
}

// NewWorld creates a new ECS world
func NewWorld() *World {
	return &World{
		entities:     make(map[EntityID]*Entity),
		components:   make(map[EntityID]ComponentMap),
		systems:      make([]System, 0),
		entityTags:   make(map[string]map[EntityID]bool),
		eventManager: NewEventManager(),
	}
}

// CreateEntity creates a new entity and adds it to the world
func (w *World) CreateEntity() *Entity {
	entity := NewEntity()
	w.entities[entity.ID] = entity
	w.components[entity.ID] = make(ComponentMap)
	return entity
}

// RemoveEntity removes an entity and all its components from the world
func (w *World) RemoveEntity(entityID EntityID) {
	if entity, exists := w.entities[entityID]; exists {
		// Remove entity from tag lookups
		for tag := range entity.Tags {
			delete(w.entityTags[tag], entityID)
			if len(w.entityTags[tag]) == 0 {
				delete(w.entityTags, tag)
			}
		}

		// Remove components and entity
		delete(w.components, entityID)
		delete(w.entities, entityID)
	}
}

// AddComponent adds a component to an entity
func (w *World) AddComponent(entityID EntityID, componentID ComponentID, component Component) {
	if _, exists := w.entities[entityID]; !exists {
		return
	}

	if _, exists := w.components[entityID]; !exists {
		w.components[entityID] = make(ComponentMap)
	}

	w.components[entityID][componentID] = component
}

// GetComponent retrieves a component from an entity
func (w *World) GetComponent(entityID EntityID, componentID ComponentID) (Component, bool) {
	if componentMap, exists := w.components[entityID]; exists {
		component, exists := componentMap[componentID]
		return component, exists
	}
	return nil, false
}

// HasComponent checks if an entity has a specific component
func (w *World) HasComponent(entityID EntityID, componentID ComponentID) bool {
	if componentMap, exists := w.components[entityID]; exists {
		_, exists := componentMap[componentID]
		return exists
	}
	return false
}

// RemoveComponent removes a component from an entity
func (w *World) RemoveComponent(entityID EntityID, componentID ComponentID) {
	if componentMap, exists := w.components[entityID]; exists {
		delete(componentMap, componentID)
	}
}

// AddSystem adds a system to the world
func (w *World) AddSystem(system System) {
	w.systems = append(w.systems, system)
}

// Update updates all systems in the world
func (w *World) Update(dt float64) {
	// Run all systems - we'll handle map-specific processing in each system
	for _, system := range w.systems {
		system.Update(w, dt)
	}
}

// GetSystems returns all systems registered in the world
func (w *World) GetSystems() []System {
	return w.systems
}

// TagEntity adds a tag to an entity and updates the tag lookup
func (w *World) TagEntity(entityID EntityID, tag string) {
	entity, exists := w.entities[entityID]
	if !exists {
		return
	}

	entity.AddTag(tag)

	// Update tag lookup
	if _, exists := w.entityTags[tag]; !exists {
		w.entityTags[tag] = make(map[EntityID]bool)
	}

	w.entityTags[tag][entityID] = true
}

// GetEntitiesWithTag returns all entities with a specific tag
func (w *World) GetEntitiesWithTag(tag string) []*Entity {
	entities := make([]*Entity, 0)

	if taggedEntities, exists := w.entityTags[tag]; exists {
		for entityID := range taggedEntities {
			if entity, ok := w.entities[entityID]; ok {
				entities = append(entities, entity)
			}
		}
	}

	return entities
}

// GetAllEntities returns a slice of all entities in the world
func (w *World) GetAllEntities() []*Entity {
	entities := make([]*Entity, 0, len(w.entities))
	for _, entity := range w.entities {
		entities = append(entities, entity)
	}
	return entities
}

// GetEventManager returns the world's event manager
func (w *World) GetEventManager() *EventManager {
	return w.eventManager
}

// EmitEvent is a convenience method to emit an event
func (w *World) EmitEvent(event Event) {
	w.eventManager.Emit(event)
}

// GetEntity returns an entity by its ID
func (w *World) GetEntity(entityID EntityID) *Entity {
	entity, exists := w.entities[entityID]
	if !exists {
		return nil
	}
	return entity
}

// GetEntitiesWithComponent returns all entities that have a specific component
func (w *World) GetEntitiesWithComponent(componentID ComponentID) []*Entity {
	entities := make([]*Entity, 0)

	for id, componentMap := range w.components {
		if _, hasComponent := componentMap[componentID]; hasComponent {
			if entity, ok := w.entities[id]; ok {
				entities = append(entities, entity)
			}
		}
	}

	return entities
}
