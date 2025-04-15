package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
	"fmt"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

// Update handles container interactions
func (s *ContainerSystem) Update(world *ecs.World, dt float64) {
	log.Printf("ContainerSystem Update called")

	// Get player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		log.Printf("No player entity found")
		return
	}
	player := playerEntities[0]

	// Get player position
	posComp, exists := world.GetComponent(player.ID, components.Position)
	if !exists {
		return
	}
	pos := posComp.(*components.PositionComponent)

	// Check for adjacent containers
	for _, entity := range world.GetEntitiesWithComponent(components.Container) {
		containerPosComp, exists := world.GetComponent(entity.ID, components.Position)
		if !exists {
			continue
		}
		containerPos := containerPosComp.(*components.PositionComponent)

		if s.isAdjacent(pos.X, pos.Y, containerPos.X, containerPos.Y) {
			// Debug log when player is adjacent to container
			log.Printf("Player adjacent to container at (%d,%d)", containerPos.X, containerPos.Y)

			// Check for E key press to interact with container
			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				log.Printf("E key pressed, attempting to open container")
				// Handle container interaction
				s.handleContainerInteraction(entity)
			}
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
