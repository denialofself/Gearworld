package components

import (
	"ebiten-rogue/ecs"
	"image/color"
)

// PositionComponent stores entity position
type PositionComponent struct {
	X, Y int
}

// RenderableComponent stores rendering information
type RenderableComponent struct {
	Char       rune        // The character in the tileset (for ASCII-based tiles)
	TileX      int         // X position in the tileset (for direct position access)
	TileY      int         // Y position in the tileset (for direct position access)
	UseTilePos bool        // Whether to use tile position instead of Char
	FG         color.Color // Foreground color
	BG         color.Color // Background color (optional)
}

// NewRenderableComponent creates a renderable component using a character code
func NewRenderableComponent(glyph rune, fg color.Color) *RenderableComponent {
	return &RenderableComponent{
		Char:       glyph,
		UseTilePos: false,
		FG:         fg,
		BG:         color.RGBA{0, 0, 0, 255}, // Default black background
	}
}

// NewRenderableComponentByPos creates a renderable component using direct position in the tileset
func NewRenderableComponentByPos(tileX, tileY int, fg color.Color) *RenderableComponent {
	return &RenderableComponent{
		TileX:      tileX,
		TileY:      tileY,
		UseTilePos: true,
		FG:         fg,
		BG:         color.RGBA{0, 0, 0, 255}, // Default black background
	}
}

// PlayerComponent indicates that an entity is controlled by the player
type PlayerComponent struct{}

// StatsComponent stores entity stats
type StatsComponent struct {
	Health          int
	MaxHealth       int
	Attack          int
	Defense         int
	Level           int
	Exp             int
	Recovery        int // Recovery points for action point regeneration
	ActionPoints    int // Current action points
	MaxActionPoints int // Maximum action points
	HealingFactor   int // Healing factor for health regeneration
}

// CollisionComponent indicates entity can collide with other entities
type CollisionComponent struct {
	Blocks bool // Whether this entity blocks movement
}

// AIComponent stores AI behavior information
type AIComponent struct {
	Type             string     // Type of AI: "random", "chase", "slow_chase", etc.
	SightRange       int        // How far the entity can see
	Target           uint64     // Target entity ID (usually the player)
	Path             []PathNode // Current path to target (if pathfinding)
	LastKnownTargetX int        // Last known X position of target
	LastKnownTargetY int        // Last known Y position of target
}

// PathNode represents a single point in a path
type PathNode struct {
	X, Y int
}

// CameraComponent tracks the viewport position for map scrolling
type CameraComponent struct {
	X, Y   int    // Top-left position of the camera in the world
	Target uint64 // Entity ID that the camera follows (usually the player)
}

// NewCameraComponent creates a new camera component that follows the specified target
func NewCameraComponent(targetEntityID uint64) *CameraComponent {
	return &CameraComponent{
		X:      0,
		Y:      0,
		Target: targetEntityID,
	}
}

// InventoryComponent represents an entity's inventory of items
type InventoryComponent struct {
	Items       []ecs.EntityID // Items in the inventory
	MaxCapacity int            // Maximum number of items the inventory can hold
}

// NewInventoryComponent creates a new inventory component with a given capacity
func NewInventoryComponent(capacity int) *InventoryComponent {
	return &InventoryComponent{
		Items:       make([]ecs.EntityID, 0),
		MaxCapacity: capacity,
	}
}

// AddItem adds an item to the inventory if there's space
// Returns true if the item was added, false if inventory is full
func (i *InventoryComponent) AddItem(itemID ecs.EntityID) bool {
	if len(i.Items) >= i.MaxCapacity {
		return false
	}

	i.Items = append(i.Items, itemID)
	return true
}

// RemoveItem removes an item from the inventory by its entity ID
// Returns true if item was found and removed, false otherwise
func (i *InventoryComponent) RemoveItem(itemID ecs.EntityID) bool {
	for idx, id := range i.Items {
		if id == itemID {
			// Remove the item by replacing it with the last element and truncating
			i.Items[idx] = i.Items[len(i.Items)-1]
			i.Items = i.Items[:len(i.Items)-1]
			return true
		}
	}
	return false
}

// GetItemByIndex returns the item at the given index or 0 if index is out of bounds
func (i *InventoryComponent) GetItemByIndex(index int) ecs.EntityID {
	if index < 0 || index >= len(i.Items) {
		return 0
	}
	return i.Items[index]
}

// HasSpace returns true if there's still room in the inventory
func (i *InventoryComponent) HasSpace() bool {
	return len(i.Items) < i.MaxCapacity
}

// IsFull returns true if the inventory is at capacity
func (i *InventoryComponent) IsFull() bool {
	return len(i.Items) >= i.MaxCapacity
}

// Size returns the current number of items in the inventory
func (i *InventoryComponent) Size() int {
	return len(i.Items)
}

// ItemComponent indicates that an entity is an item that can be collected
type ItemComponent struct {
	ItemType    string      // Type of item: "weapon", "armor", "potion", etc.
	Value       int         // Base value/power of the item
	Weight      int         // Weight of the item (for inventory capacity calculations)
	Description string      // Description of the item
	TemplateID  string      // ID of the template that created this item
	Data        interface{} // Additional item-specific data
}

// NewItemComponent creates a new item component
func NewItemComponent(itemType string, value int, weight int) *ItemComponent {
	return &ItemComponent{
		ItemType:    itemType,
		Value:       value,
		Weight:      weight,
		Description: "",
		TemplateID:  "",
		Data:        nil,
	}
}

// NewItemComponentFromTemplate creates a new item component from a template
func NewItemComponentFromTemplate(templateID string, itemType string, value int, weight int, description string) *ItemComponent {
	return &ItemComponent{
		ItemType:    itemType,
		Value:       value,
		Weight:      weight,
		Description: description,
		TemplateID:  templateID,
		Data:        nil,
	}
}

// FOVComponent represents an entity's field of vision capabilities
type FOVComponent struct {
	Range       int  // How far the entity can see in tiles
	LightSource bool // Whether this entity emits light
	LightRange  int  // How far the light reaches if this is a light source
}

// NewFOVComponent creates a new FOV component with the specified range
func NewFOVComponent(visionRange int) *FOVComponent {
	return &FOVComponent{
		Range:       visionRange,
		LightSource: false,
		LightRange:  0,
	}
}

// NewLightSourceFOVComponent creates a new FOV component for a light source
func NewLightSourceFOVComponent(visionRange, lightRange int) *FOVComponent {
	return &FOVComponent{
		Range:       visionRange,
		LightSource: true,
		LightRange:  lightRange,
	}
}

// EquipmentSlot defines possible equipment slots
type EquipmentSlot string

const (
	SlotHead      EquipmentSlot = "head"
	SlotBody      EquipmentSlot = "body"
	SlotMainHand  EquipmentSlot = "mainhand"
	SlotOffHand   EquipmentSlot = "offhand"
	SlotFeet      EquipmentSlot = "feet"
	SlotAccessory EquipmentSlot = "accessory"
)

// ItemEffect represents an effect that an item can have on an entity
type ItemEffect struct {
	Component string      // Component name to affect
	Property  string      // Property name within the component
	Operation string      // Operation to perform: "add", "multiply", "set", etc.
	Value     interface{} // Value to apply in the operation
}

// EquipmentComponent represents equipped items
type EquipmentComponent struct {
	EquippedItems map[EquipmentSlot]ecs.EntityID // Map of slot to item entity ID
	ActiveEffects map[ecs.EntityID][]ItemEffect  // Map of item entity ID to active effects
}

// NewEquipmentComponent creates a new equipment component
func NewEquipmentComponent() *EquipmentComponent {
	return &EquipmentComponent{
		EquippedItems: make(map[EquipmentSlot]ecs.EntityID),
		ActiveEffects: make(map[ecs.EntityID][]ItemEffect),
	}
}

// IsSlotOccupied checks if a slot is currently occupied
func (e *EquipmentComponent) IsSlotOccupied(slot EquipmentSlot) bool {
	_, occupied := e.EquippedItems[slot]
	return occupied
}

// GetEquippedItem returns the item ID equipped in a slot, or 0 if empty
func (e *EquipmentComponent) GetEquippedItem(slot EquipmentSlot) ecs.EntityID {
	if itemID, ok := e.EquippedItems[slot]; ok {
		return itemID
	}
	return 0
}

// EquipItem equips an item in a slot
func (e *EquipmentComponent) EquipItem(slot EquipmentSlot, itemID ecs.EntityID) {
	e.EquippedItems[slot] = itemID
}

// UnequipItem removes an item from a slot
func (e *EquipmentComponent) UnequipItem(slot EquipmentSlot) ecs.EntityID {
	if itemID, ok := e.EquippedItems[slot]; ok {
		delete(e.EquippedItems, slot)
		return itemID
	}
	return 0
}

// AddEffect adds an effect for an item
func (e *EquipmentComponent) AddEffect(itemID ecs.EntityID, effect ItemEffect) {
	if _, ok := e.ActiveEffects[itemID]; !ok {
		e.ActiveEffects[itemID] = make([]ItemEffect, 0)
	}
	e.ActiveEffects[itemID] = append(e.ActiveEffects[itemID], effect)
}

// RemoveEffects removes all effects for an item
func (e *EquipmentComponent) RemoveEffects(itemID ecs.EntityID) {
	delete(e.ActiveEffects, itemID)
}

// GetAllEffects returns all active effects
func (e *EquipmentComponent) GetAllEffects() []ItemEffect {
	allEffects := make([]ItemEffect, 0)
	for _, effects := range e.ActiveEffects {
		allEffects = append(allEffects, effects...)
	}
	return allEffects
}

// ContainerComponent represents a container that can hold items
type ContainerComponent struct {
	Items       []ecs.EntityID
	MaxCapacity int
	Locked      bool
	KeyID       string
	Looted      bool // Track if the container has been looted
}

// NewContainerComponent creates a new container component
func NewContainerComponent(capacity int) *ContainerComponent {
	return &ContainerComponent{
		Items:       make([]ecs.EntityID, 0),
		MaxCapacity: capacity,
		Locked:      false,
		KeyID:       "",
		Looted:      false,
	}
}

// AddItem adds an item to the container
func (c *ContainerComponent) AddItem(itemID ecs.EntityID) bool {
	if len(c.Items) >= c.MaxCapacity {
		return false
	}
	c.Items = append(c.Items, itemID)
	return true
}

// RemoveItem removes an item from the container
func (c *ContainerComponent) RemoveItem(itemID ecs.EntityID) bool {
	for i, id := range c.Items {
		if id == itemID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return true
		}
	}
	return false
}
