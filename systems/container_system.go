package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
	"image/color"
	"log"
	"strings"
)

// ContainerSystem handles container-related logic
type ContainerSystem struct {
	world *ecs.World
}

// NewContainerSystem creates a new container system
func NewContainerSystem(world *ecs.World) *ContainerSystem {
	return &ContainerSystem{
		world: world,
	}
}

// Update is called every frame but only processes container interactions during player turns
func (s *ContainerSystem) Update(world *ecs.World, dt float64) {
	// No processing needed every frame in a turn-based game
}

// HandleEvent processes container-related events
func (s *ContainerSystem) HandleEvent(world *ecs.World, event ecs.Event) {
	switch e := event.(type) {
	case ExamineEvent:
		// Check if the examined entity is a container
		container := s.world.GetEntity(e.TargetID)
		if container != nil && container.HasTag("container") {
			// Get player position
			playerEntities := s.world.GetEntitiesWithTag("player")
			if len(playerEntities) == 0 {
				return
			}
			player := playerEntities[0]

			playerPos, exists := s.world.GetComponent(player.ID, components.Position)
			if !exists {
				return
			}
			pos := playerPos.(*components.PositionComponent)

			// Get container position
			containerPos, exists := s.world.GetComponent(container.ID, components.Position)
			if !exists {
				return
			}
			contPos := containerPos.(*components.PositionComponent)

			// Check if player is adjacent to container
			if s.isAdjacent(pos.X, pos.Y, contPos.X, contPos.Y) {
				GetDebugLog().Add(fmt.Sprintf("Examining container at (%d,%d)", contPos.X, contPos.Y))
				s.handleContainerInteraction(container)
			} else {
				GetMessageLog().AddEnvironment("You need to be next to the container to examine it.")
			}
		}
	}
}

// checkAdjacentContainers checks if the player is adjacent to any containers at the given position
func (s *ContainerSystem) checkAdjacentContainers(playerX, playerY int) {
	// Get all container entities
	containerEntities := s.world.GetEntitiesWithTag("container")

	for _, container := range containerEntities {
		// Get container position
		posComp, exists := s.world.GetComponent(container.ID, components.Position)
		if !exists {
			continue
		}
		containerPos := posComp.(*components.PositionComponent)

		// Check if player is adjacent to container
		if s.isAdjacent(playerX, playerY, containerPos.X, containerPos.Y) {
			GetDebugLog().Add(fmt.Sprintf("Player adjacent to container at (%d,%d)", containerPos.X, containerPos.Y))
			// Handle container interaction
			s.handleContainerInteraction(container)
		}
	}
}

// isAdjacent checks if two positions are adjacent (including diagonals)
func (s *ContainerSystem) isAdjacent(x1, y1, x2, y2 int) bool {
	dx := x1 - x2
	dy := y1 - y2
	return dx >= -1 && dx <= 1 && dy >= -1 && dy <= 1
}

// handleContainerInteraction handles player interaction with a container
func (s *ContainerSystem) handleContainerInteraction(container *ecs.Entity) {
	log.Printf("Handling container interaction")

	// Get container component
	containerComp, exists := s.world.GetComponent(container.ID, components.Container)
	if !exists {
		log.Printf("Container component not found")
		return
	}
	containerData := containerComp.(*components.ContainerComponent)

	// Check if container is locked
	if containerData.Locked {
		log.Printf("Container is locked")
		// TODO: Handle locked container logic
		return
	}

	// Get container name for messages
	var containerName string = "a container"
	if nameComp, exists := s.world.GetComponent(container.ID, components.Name); exists {
		containerName = nameComp.(*components.NameComponent).Name
	}

	// If container hasn't been looted yet, show what's inside
	if !containerData.Looted {
		// Get list of items in container
		var itemNames []string
		for _, itemID := range containerData.Items {
			if nameComp, exists := s.world.GetComponent(itemID, components.Name); exists {
				itemNames = append(itemNames, nameComp.(*components.NameComponent).Name)
			}
		}

		// Show environment message about container contents
		if len(itemNames) > 0 {
			GetMessageLog().AddEnvironment(fmt.Sprintf("You open %s and find: %s", containerName, strings.Join(itemNames, ", ")))
		} else {
			GetMessageLog().AddEnvironment(fmt.Sprintf("You open %s but find nothing inside.", containerName))
		}

		// Mark container as looted
		containerData.Looted = true

		// Darken the container's appearance
		if renderComp, exists := s.world.GetComponent(container.ID, components.Renderable); exists {
			renderable := renderComp.(*components.RenderableComponent)
			// Darken the foreground color by reducing RGB values
			if fgRGBA, ok := renderable.FG.(color.RGBA); ok {
				renderable.FG = color.RGBA{
					R: uint8(float64(fgRGBA.R) * 0.5),
					G: uint8(float64(fgRGBA.G) * 0.5),
					B: uint8(float64(fgRGBA.B) * 0.5),
					A: fgRGBA.A,
				}
			}
		}
	}

	// Handle item pickup from container
	s.handleItemPickup(container)
}

// handleItemPickup handles picking up items from a container
func (s *ContainerSystem) handleItemPickup(container *ecs.Entity) {
	GetDebugLog().Add("Starting item pickup from container")

	// Get container component
	containerComp, exists := s.world.GetComponent(container.ID, components.Container)
	if !exists {
		GetDebugLog().Add("Container component not found during pickup")
		return
	}
	containerData := containerComp.(*components.ContainerComponent)

	// Get player entity
	playerEntities := s.world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		GetDebugLog().Add("Player entity not found during pickup")
		return
	}
	player := playerEntities[0]

	// Get player inventory
	inventoryComp, exists := s.world.GetComponent(player.ID, components.Inventory)
	if !exists {
		GetDebugLog().Add("Player inventory not found during pickup")
		return
	}
	inventory := inventoryComp.(*components.InventoryComponent)

	// Log initial state
	GetDebugLog().Add(fmt.Sprintf("Container has %d items before pickup", len(containerData.Items)))

	// Create a copy of the items list to avoid modifying it during iteration
	itemsToPickup := make([]ecs.EntityID, len(containerData.Items))
	copy(itemsToPickup, containerData.Items)

	// Try to pick up each item in the container
	for _, itemID := range itemsToPickup {
		// Get item name
		var itemName string = "an item"
		if nameComp, exists := s.world.GetComponent(itemID, components.Name); exists {
			itemName = nameComp.(*components.NameComponent).Name
		}

		// Get item type for debug logging
		var itemType string = "unknown"
		if itemComp, exists := s.world.GetComponent(itemID, components.Item); exists {
			itemType = itemComp.(*components.ItemComponent).ItemType
		}

		GetDebugLog().Add(fmt.Sprintf("Attempting to pick up %s (type: %s, ID: %d)", itemName, itemType, itemID))

		// Add item to player inventory
		if inventory.AddItem(itemID) {
			// Remove item from container
			containerData.RemoveItem(itemID)
			GetDebugLog().Add(fmt.Sprintf("Successfully picked up %s", itemName))
		} else {
			GetDebugLog().Add(fmt.Sprintf("Failed to add %s to inventory", itemName))
		}
	}

	// Log final state
	GetDebugLog().Add(fmt.Sprintf("Container has %d items after pickup", len(containerData.Items)))
}

// Initialize sets up event listeners for the container system
func (s *ContainerSystem) Initialize(world *ecs.World) {
	// Store world reference
	s.world = world

	// Subscribe to examine events
	world.GetEventManager().Subscribe(EventExamine, func(event ecs.Event) {
		examineEvent := event.(ExamineEvent)
		s.HandleEvent(world, examineEvent)
	})
}
