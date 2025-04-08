package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
)

// CameraSystem handles viewport positioning and scrolling
type CameraSystem struct {
}

// NewCameraSystem creates a new camera system
func NewCameraSystem() *CameraSystem {
	return &CameraSystem{}
}

// Update updates the camera position based on the player's position
func (s *CameraSystem) Update(world *ecs.World, dt float64) {
	// Find player entity
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}

	playerID := playerEntities[0].ID

	// Get the player position
	playerPosComp, hasPos := world.GetComponent(playerID, components.Position)
	if !hasPos {
		return
	}
	playerPos := playerPosComp.(*components.PositionComponent)

	// Find camera entity or create one if it doesn't exist
	cameraEntities := world.GetEntitiesWithTag("camera")
	var cameraID ecs.EntityID
	var cameraComp *components.CameraComponent

	if len(cameraEntities) == 0 {
		// Create a new camera entity
		cameraEntity := world.CreateEntity()
		cameraID = cameraEntity.ID
		cameraComp = components.NewCameraComponent(uint64(playerID))
		world.AddComponent(cameraID, components.Camera, cameraComp)
	} else {
		cameraID = cameraEntities[0].ID
		comp, _ := world.GetComponent(cameraID, components.Camera)
		cameraComp = comp.(*components.CameraComponent)
	}

	// Find map entity
	standardMapEntities := world.GetEntitiesWithTag("map")

	if len(standardMapEntities) > 0 {
		// Handle standard map camera positioning
		s.updateCameraForStandardMap(world, playerPos, cameraComp, standardMapEntities[0].ID)
	}
}

// updateCameraForStandardMap centers the camera on the player with boundary constraints
func (s *CameraSystem) updateCameraForStandardMap(world *ecs.World, playerPos *components.PositionComponent, camera *components.CameraComponent, mapID ecs.EntityID) {
	mapComp, hasMap := world.GetComponent(mapID, components.MapComponentID)
	if !hasMap {
		return
	}
	mapData := mapComp.(*components.MapComponent)

	// Calculate ideal camera position (center player in viewport)
	// Subtract half the screen width and height to center the player
	idealCameraX := playerPos.X - config.GameScreenWidth/2
	idealCameraY := playerPos.Y - config.GameScreenHeight/2

	// Constrain camera to map boundaries
	if idealCameraX < 0 {
		idealCameraX = 0
	} else if idealCameraX > mapData.Width-config.GameScreenWidth {
		idealCameraX = mapData.Width - config.GameScreenWidth
	}

	if idealCameraY < 0 {
		idealCameraY = 0
	} else if idealCameraY > mapData.Height-config.GameScreenHeight {
		idealCameraY = mapData.Height - config.GameScreenHeight
	}

	// Update camera position with smooth following
	// Simple camera smoothing can be added here if desired
	camera.X = idealCameraX
	camera.Y = idealCameraY
}

// WorldToScreen converts world coordinates to screen coordinates
func (s *CameraSystem) WorldToScreen(world *ecs.World, worldX, worldY int) (screenX, screenY int) {
	cameraEntities := world.GetEntitiesWithTag("camera")
	if len(cameraEntities) == 0 {
		// If no camera, just pass through the coordinates
		return worldX, worldY
	}

	var camera *components.CameraComponent
	if comp, exists := world.GetComponent(cameraEntities[0].ID, components.Camera); exists {
		camera = comp.(*components.CameraComponent)
	} else {
		return worldX, worldY
	}

	// Convert world coordinates to screen coordinates
	screenX = worldX - camera.X
	screenY = worldY - camera.Y

	return screenX, screenY
}

// ScreenToWorld converts screen coordinates to world coordinates
func (s *CameraSystem) ScreenToWorld(world *ecs.World, screenX, screenY int) (worldX, worldY int) {
	cameraEntities := world.GetEntitiesWithTag("camera")
	if len(cameraEntities) == 0 {
		// If no camera, just pass through the coordinates
		return screenX, screenY
	}

	var camera *components.CameraComponent
	if comp, exists := world.GetComponent(cameraEntities[0].ID, components.Camera); exists {
		camera = comp.(*components.CameraComponent)
	} else {
		return screenX, screenY
	}

	// Convert screen coordinates to world coordinates
	worldX = screenX + camera.X
	worldY = screenY + camera.Y

	return worldX, worldY
}

// IsVisible checks if a world position is visible on screen
func (s *CameraSystem) IsVisible(world *ecs.World, worldX, worldY int) bool {
	cameraEntities := world.GetEntitiesWithTag("camera")
	if len(cameraEntities) == 0 {
		return true // If no camera, assume everything is visible
	}

	var camera *components.CameraComponent
	if comp, exists := world.GetComponent(cameraEntities[0].ID, components.Camera); exists {
		camera = comp.(*components.CameraComponent)
	} else {
		return true
	}

	// Check if the position is within the camera's view
	return worldX >= camera.X &&
		worldX < camera.X+config.GameScreenWidth &&
		worldY >= camera.Y &&
		worldY < camera.Y+config.GameScreenHeight
}
