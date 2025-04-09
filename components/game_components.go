package components

import (
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
