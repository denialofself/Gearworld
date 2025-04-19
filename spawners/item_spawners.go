package spawners

import (
	"ebiten-rogue/components"
	"ebiten-rogue/data"
	"ebiten-rogue/ecs"
	"ebiten-rogue/systems"
	"fmt"
	"image/color"
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
// If addToContainer is true, position components will not be added
// If templateID is empty, it will create a basic item using the provided parameters
func (s *ItemSpawner) CreateItem(x, y int, templateID string, addToContainer bool, opts ...ItemOption) (*ecs.Entity, error) {
	// Create the item entity
	itemEntity := s.world.CreateEntity()
	itemEntity.AddTag("item")
	s.world.TagEntity(itemEntity.ID, "item")

	var itemComp *components.ItemComponent
	var itemColor color.Color
	var tileX, tileY int
	var itemName string

	if templateID != "" {
		// Try to load the item template
		template, exists := s.templateManager.GetItemTemplate(templateID)
		if !exists {
			return nil, fmt.Errorf("no item template found with ID '%s'", templateID)
		}

		// Add any additional tags from the template
		for _, tag := range template.Tags {
			s.world.TagEntity(itemEntity.ID, tag)
		}

		// Get visual properties from template
		itemColor = data.ParseHexColor(template.Color)
		tileX = template.TileX
		tileY = template.TileY
		itemName = template.Name

		// Create the item component from template
		itemComp = components.NewItemComponentFromTemplate(
			template.ID,
			template.ItemType,
			template.Value,
			template.Weight,
			template.Description,
		)

		// Add name component early
		s.world.AddComponent(itemEntity.ID, components.Name, components.NewNameComponent(itemName))

		// If item has effects, process them
		if len(template.Effects) > 0 {
			effects := make([]components.GameEffect, 0, len(template.Effects))

			// Convert each effect from map to GameEffect struct
			for _, effectMap := range template.Effects {
				// For equipment items, always set the type to EffectTypeEquipment
				effectType := components.EffectType(effectMap["type"].(string))
				if itemComp.ItemType == "weapon" || itemComp.ItemType == "armor" ||
					itemComp.ItemType == "headgear" || itemComp.ItemType == "shield" ||
					itemComp.ItemType == "boots" || itemComp.ItemType == "accessory" {
					effectType = components.EffectTypeEquipment
				}

				effect := components.GameEffect{
					Type:      effectType,
					Operation: components.EffectOperation(effectMap["operation"].(string)),
					Value:     effectMap["value"].(float64),
					Duration:  int(effectMap["duration"].(float64)),
					Source:    itemEntity.ID,
					Target: struct {
						Component string
						Property  string
					}{
						Component: effectMap["target"].(map[string]interface{})["component"].(string),
						Property:  effectMap["target"].(map[string]interface{})["property"].(string),
					},
				}
				effects = append(effects, effect)
			}

			// Store the effects in the item's Data field
			itemComp.Data = effects
		}
	} else {
		// Apply any provided options
		options := defaultItemOptions()
		for _, opt := range opts {
			opt(options)
		}

		// Set visual properties based on item type
		switch options.itemType {
		case "weapon":
			tileX, tileY = 15, 7 // Sword
			itemColor = color.RGBA{220, 220, 255, 255}
		case "armor":
			tileX, tileY = 15, 9 // Armor
			itemColor = color.RGBA{200, 200, 220, 255}
		case "potion":
			tileX, tileY = 15, 3 // Potion
			itemColor = color.RGBA{255, 100, 100, 255}
		case "scroll":
			tileX, tileY = 15, 0 // Scroll
			itemColor = color.RGBA{255, 255, 200, 255}
		default:
			tileX, tileY = 12, 0 // Default item
			itemColor = color.RGBA{200, 200, 200, 255}
		}

		// Create basic item component
		itemComp = components.NewItemComponent(
			options.itemType,
			options.value,
			options.weight,
		)

		// Add name component early
		s.world.AddComponent(itemEntity.ID, components.Name, components.NewNameComponent(options.name))
	}

	// Only add position and renderable components if not being added to a container
	if !addToContainer {
		// Add position component
		s.world.AddComponent(itemEntity.ID, components.Position, &components.PositionComponent{
			X: x,
			Y: y,
		})

		// Add renderable component
		s.world.AddComponent(itemEntity.ID, components.Renderable, components.NewRenderableComponentByPos(
			tileX, tileY,
			itemColor,
		))
	}

	// Add the item component
	s.world.AddComponent(itemEntity.ID, components.Item, itemComp)

	// Add map context component if spawnMapID is set
	if s.spawnMapID != 0 {
		s.world.AddComponent(itemEntity.ID, components.MapContextID, components.NewMapContextComponent(s.spawnMapID))
	}

	return itemEntity, nil
}

// ItemOptions holds optional parameters for item creation
type ItemOptions struct {
	name     string
	itemType string
	value    int
	weight   int
}

// ItemOption is a function that modifies ItemOptions
type ItemOption func(*ItemOptions)

// defaultItemOptions returns default item options
func defaultItemOptions() *ItemOptions {
	return &ItemOptions{
		name:     "Unknown Item",
		itemType: "miscellaneous",
		value:    1,
		weight:   1,
	}
}

// WithName sets the item name
func WithName(name string) ItemOption {
	return func(o *ItemOptions) {
		o.name = name
	}
}

// WithType sets the item type
func WithType(itemType string) ItemOption {
	return func(o *ItemOptions) {
		o.itemType = itemType
	}
}

// WithValue sets the item value
func WithValue(value int) ItemOption {
	return func(o *ItemOptions) {
		o.value = value
	}
}

// WithWeight sets the item weight
func WithWeight(weight int) ItemOption {
	return func(o *ItemOptions) {
		o.weight = weight
	}
}
