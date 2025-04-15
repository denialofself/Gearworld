package spawners

import (
	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/systems"
	"fmt"
)

// ItemSpawner handles the creation of items and containers
type ItemSpawner struct {
	world           *ecs.World
	templateManager *data.EntityTemplateManager
	spawnMapID      ecs.EntityID
}

// NewItemSpawner creates a new item spawner
func NewItemSpawner(world *ecs.World, templateManager *data.EntityTemplateManager) *ItemSpawner {
	return &ItemSpawner{
		world:           world,
		templateManager: templateManager,
	}
}

// SetSpawnMapID sets the map ID for spawned items
func (s *ItemSpawner) SetSpawnMapID(mapID ecs.EntityID) {
	s.spawnMapID = mapID
}

// CreateContainer creates a container from a template
func (s *ItemSpawner) CreateContainer(x, y int, templateID string) (*ecs.Entity, error) {
	// Get the container template
	template, exists := s.templateManager.GetContainerTemplate(templateID)
	if !exists {
		return nil, fmt.Errorf("no container template found with ID '%s'", templateID)
	}

	// Create the container entity
	container := s.world.CreateEntity()
	container.AddTag("container")
	s.world.TagEntity(container.ID, "container")

	// Add position component
	s.world.AddComponent(container.ID, components.Position, &components.PositionComponent{
		X: x,
		Y: y,
	})

	// Add renderable component using template data
	containerColor := data.ParseHexColor(template.Color)
	s.world.AddComponent(container.ID, components.Renderable, components.NewRenderableComponentByPos(
		template.TileX, template.TileY,
		containerColor,
	))

	// Create the container component
	containerComp := components.NewContainerComponent(template.Capacity)
	containerComp.Locked = template.Locked

	// Add initial items if specified
	itemsCreated := 0
	for _, initialItem := range template.InitialItems {
		systems.GetDebugLog().Add(fmt.Sprintf("Processing initial item entry: template_id=%s, count=%d", initialItem.TemplateID, initialItem.Count))
		for i := 0; i < initialItem.Count; i++ {
			// Create item (position doesn't matter since it's going in container)
			item, err := s.CreateItem(0, 0, initialItem.TemplateID, true)
			if err != nil {
				systems.GetDebugLog().Add(fmt.Sprintf("Failed to create item %s: %v", initialItem.TemplateID, err))
				continue
			}

			// Try to add the item to the container
			if !containerComp.AddItem(item.ID) {
				systems.GetDebugLog().Add(fmt.Sprintf("Failed to add item %s to container: container full", initialItem.TemplateID))
				s.world.RemoveEntity(item.ID)
				continue
			}

			itemsCreated++
			systems.GetDebugLog().Add(fmt.Sprintf("Successfully added item %s (ID: %d) to container", initialItem.TemplateID, item.ID))
		}
	}

	systems.GetDebugLog().Add(fmt.Sprintf("Created container with %d items total", itemsCreated))

	// Add the container component
	s.world.AddComponent(container.ID, components.Container, containerComp)

	// Add name component
	s.world.AddComponent(container.ID, components.Name, components.NewNameComponent(template.Name))

	// Add map context component if spawnMapID is set
	if s.spawnMapID != 0 {
		s.world.AddComponent(container.ID, components.MapContextID, components.NewMapContextComponent(s.spawnMapID))
	}

	return container, nil
}

// CreateItem creates an item entity that can be collected by the player
func (s *ItemSpawner) CreateItem(x, y int, itemTemplateID string, addToContainer bool) (*ecs.Entity, error) {
	// Try to load the item template
	template, exists := s.templateManager.GetItemTemplate(itemTemplateID)
	if !exists {
		return nil, fmt.Errorf("no item template found with ID '%s'", itemTemplateID)
	}

	// Create the item entity
	itemEntity := s.world.CreateEntity()
	itemEntity.AddTag("item")
	s.world.TagEntity(itemEntity.ID, "item")

	// Add any additional tags from the template
	for _, tag := range template.Tags {
		s.world.TagEntity(itemEntity.ID, tag)
	}

	// Only add position and renderable components if the item is not being added to a container
	if !addToContainer {
		// Add position component
		s.world.AddComponent(itemEntity.ID, components.Position, &components.PositionComponent{
			X: x,
			Y: y,
		})

		// Add renderable component using template data
		itemColor := data.ParseHexColor(template.Color)
		s.world.AddComponent(itemEntity.ID, components.Renderable, components.NewRenderableComponentByPos(
			template.TileX, template.TileY,
			itemColor,
		))
	}

	// Create the item component
	itemComp := components.NewItemComponentFromTemplate(
		template.ID,
		template.ItemType,
		template.Value,
		template.Weight,
		template.Description,
	)

	// If item has effects, process them
	if len(template.Effects) > 0 {
		effects := make([]components.ItemEffect, 0, len(template.Effects))

		// Convert each effect from map to ItemEffect struct
		for _, effectMap := range template.Effects {
			effect := components.ItemEffect{
				Component: effectMap["component"].(string),
				Property:  effectMap["property"].(string),
				Operation: effectMap["operation"].(string),
				Value:     effectMap["value"],
			}
			effects = append(effects, effect)
		}

		// Store the effects in the item's Data field
		itemComp.Data = effects
	}

	// Add the item component
	s.world.AddComponent(itemEntity.ID, components.Item, itemComp)

	// Add name component
	s.world.AddComponent(itemEntity.ID, components.Name, components.NewNameComponent(template.Name))

	// Add map context component if spawnMapID is set
	if s.spawnMapID != 0 {
		s.world.AddComponent(itemEntity.ID, components.MapContextID, components.NewMapContextComponent(s.spawnMapID))
	}

	return itemEntity, nil
}
