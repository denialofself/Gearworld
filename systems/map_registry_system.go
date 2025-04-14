package systems

import (
	"ebiten-rogue/components"
	"ebiten-rogue/config"
	"ebiten-rogue/ecs"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// MapRegistrySystem manages multiple maps and transitioning between them
type MapRegistrySystem struct {
	world                *ecs.World
	maps                 map[string][]*ecs.Entity      // Maps stored by type and level: e.g. "worldmap_0", "dungeon_1"
	activeMapID          ecs.EntityID                  // The currently active map
	lastMapID            ecs.EntityID                  // The previous map (to return to)
	lastPosition         *components.PositionComponent // Last position in previous map
	transitionInProgress bool                          // Flag to indicate a transition is in progress
}

// NewMapRegistrySystem creates a new map registry system
func NewMapRegistrySystem() *MapRegistrySystem {
	return &MapRegistrySystem{
		maps:                 make(map[string][]*ecs.Entity),
		transitionInProgress: false,
	}
}

// Initialize sets up the map registry system
func (s *MapRegistrySystem) Initialize(world *ecs.World) {
	s.world = world
}

// Update checks for map transitions and handles them
func (s *MapRegistrySystem) Update(world *ecs.World, dt float64) {
	// Store world reference
	s.world = world

	// Check for map transition events
	s.handleMapTransitions(world)
}

// RegisterMap adds a map to the registry or replaces an existing one
func (s *MapRegistrySystem) RegisterMap(mapEntity *ecs.Entity) {
	// Get the map type component
	mapTypeComp, exists := s.world.GetComponent(mapEntity.ID, components.MapType)
	if !exists {
		GetMessageLog().Add("Error: Map type component not found when registering map")
		return
	}

	mapType := mapTypeComp.(*components.MapTypeComponent)
	mapKey := s.generateMapKey(mapType.MapType, mapType.Level)

	// Check if we already have maps of this type/level
	if _, exists := s.maps[mapKey]; !exists {
		s.maps[mapKey] = make([]*ecs.Entity, 0)
	}

	// Add the map to our registry
	s.maps[mapKey] = append(s.maps[mapKey], mapEntity)

	// Log the registration event with detailed ID information
	GetDebugLog().Add(fmt.Sprintf("REGISTRY: Registered %s (Level %d) with ID: %d",
		mapType.MapType, mapType.Level, mapEntity.ID))
}

// SetActiveMap sets the currently active map
func (s *MapRegistrySystem) SetActiveMap(mapEntity *ecs.Entity) {
	// Store last map and position
	if s.activeMapID != 0 {
		s.lastMapID = s.activeMapID

		// Store player's position on the map they're leaving
		player := s.getPlayer()
		if player != nil {
			posComp, exists := s.world.GetComponent(player.ID, components.Position)
			if exists {
				s.lastPosition = &components.PositionComponent{
					X: posComp.(*components.PositionComponent).X,
					Y: posComp.(*components.PositionComponent).Y,
				}
			}
		}
	}

	// Update active map ID
	s.activeMapID = mapEntity.ID

	// Get map type info for better logging
	var mapType string = "unknown"
	var mapLevel int = -1
	if typeComp, exists := s.world.GetComponent(mapEntity.ID, components.MapType); exists {
		mapTypeComp := typeComp.(*components.MapTypeComponent)
		mapType = mapTypeComp.MapType
		mapLevel = mapTypeComp.Level
	}

	GetDebugLog().Add(fmt.Sprintf("SET ACTIVE MAP: Setting active map to %s (Level %d, ID: %d)",
		mapType, mapLevel, mapEntity.ID))

	// Notify the map system of the change
	mapSystem := s.getMapSystem()
	if mapSystem != nil {
		mapSystem.SetActiveMap(mapEntity)
		GetDebugLog().Add(fmt.Sprintf("SET ACTIVE MAP: Propagated to MapSystem"))
	} else {
		GetDebugLog().Add("ERROR: Could not find MapSystem to propagate active map change")
	}

	// Double-check that the update was successful
	if s.activeMapID != mapEntity.ID {
		GetDebugLog().Add(fmt.Sprintf("ERROR: Failed to set active map ID - expected %d, got %d",
			mapEntity.ID, s.activeMapID))
	}
}

// GetActiveMap returns the currently active map entity
func (s *MapRegistrySystem) GetActiveMap() *ecs.Entity {
	if s.activeMapID == 0 {
		return nil
	}
	return s.world.GetEntity(s.activeMapID)
}

// GetLastMap returns the previously active map entity
func (s *MapRegistrySystem) GetLastMap() *ecs.Entity {
	if s.lastMapID == 0 {
		return nil
	}
	return s.world.GetEntity(s.lastMapID)
}

// GetLastPosition returns the position in the previous map
func (s *MapRegistrySystem) GetLastPosition() *components.PositionComponent {
	return s.lastPosition
}

// GetMapByType returns a map of the specified type and level
func (s *MapRegistrySystem) GetMapByType(mapType string, level int) *ecs.Entity {
	mapKey := s.generateMapKey(mapType, level)
	maps, exists := s.maps[mapKey]
	if !exists || len(maps) == 0 {
		return nil
	}
	return maps[0]
}

// generateMapKey creates a key for the maps registry
func (s *MapRegistrySystem) generateMapKey(mapType string, level int) string {
	return fmt.Sprintf("%s_%d", mapType, level)
}

// getMapSystem retrieves the MapSystem from the world
func (s *MapRegistrySystem) getMapSystem() *MapSystem {
	systems := s.world.GetSystems()
	for _, system := range systems {
		if mapSys, ok := system.(*MapSystem); ok {
			return mapSys
		}
	}
	return nil
}

// getPlayer returns the player entity
func (s *MapRegistrySystem) getPlayer() *ecs.Entity {
	playerEntities := s.world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return nil
	}
	return playerEntities[0]
}

// handleMapTransitions processes transitions between maps when player interacts with stairs
func (s *MapRegistrySystem) handleMapTransitions(world *ecs.World) {
	// Direct console output to verify this function is being called
	fmt.Println("TRANSITION CHECK: handleMapTransitions called")

	// If a transition is in progress, don't allow another one to start
	if s.transitionInProgress {
		GetDebugLog().Add("TRANSITION: Ignoring new transition request - a transition is already in progress")
		fmt.Println("TRANSITION: Already in progress")
		return
	}

	player := s.getPlayer()
	if player == nil {
		fmt.Println("TRANSITION CHECK: No player found")
		return
	}

	// Get player position
	posCompInterface, exists := world.GetComponent(player.ID, components.Position)
	if !exists {
		fmt.Println("TRANSITION CHECK: No position component found")
		return
	}
	playerPos := posCompInterface.(*components.PositionComponent)

	// Get currently active map
	activeMap := s.GetActiveMap()
	if activeMap == nil {
		fmt.Println("TRANSITION CHECK: No active map found")
		return
	}

	mapCompInterface, exists := world.GetComponent(activeMap.ID, components.MapComponentID)
	if !exists {
		fmt.Println("TRANSITION CHECK: No map component found")
		return
	}
	mapComp := mapCompInterface.(*components.MapComponent)

	// Check if player is standing on stairs
	if playerPos.X < 0 || playerPos.Y < 0 || playerPos.X >= mapComp.Width || playerPos.Y >= mapComp.Height {
		fmt.Println("TRANSITION CHECK: Player out of bounds")
		return
	}

	tileUnderPlayer := mapComp.Tiles[playerPos.Y][playerPos.X]
	fmt.Printf("TRANSITION CHECK: Player at (%d,%d) on tile type %d\n", playerPos.X, playerPos.Y, tileUnderPlayer)

	// Check for up or down stairs
	isStairsUp := tileUnderPlayer == components.TileStairsUp
	isStairsDown := tileUnderPlayer == components.TileStairsDown

	// If player is on stairs, check for transition input
	if (isStairsUp || isStairsDown) && ebiten.IsKeyPressed(ebiten.KeyEnter) {
		fmt.Printf("TRANSITION TRIGGERED: Player at (%d,%d) on %s pressed ENTER\n",
			playerPos.X, playerPos.Y,
			map[bool]string{true: "Up Stairs", false: "Down Stairs"}[isStairsUp])

		// Log the transition attempt
		GetDebugLog().Add(fmt.Sprintf("TRANSITION TRIGGERED: Player at (%d,%d) on tile type %d pressed ENTER",
			playerPos.X, playerPos.Y, tileUnderPlayer))

		// Start the transition
		s.transitionBetweenMaps(world, tileUnderPlayer, playerPos)
	} else if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		fmt.Printf("ENTER pressed but player not on stairs (tile type: %d)\n", tileUnderPlayer)
	}
}

// transitionBetweenMaps handles player movement between maps
func (s *MapRegistrySystem) transitionBetweenMaps(world *ecs.World, tileType int, playerPos *components.PositionComponent) {
	// Set the transition flag to prevent sync operations
	s.transitionInProgress = true
	fmt.Println("=====================================")
	fmt.Println("TRANSITION: STARTING MAP TRANSITION")
	fmt.Printf("TRANSITION: Player on tile type %d\n", tileType)
	fmt.Println("=====================================")

	GetDebugLog().Add("=====================================")
	GetDebugLog().Add("TRANSITION: STARTING MAP TRANSITION")
	GetDebugLog().Add("=====================================")

	// Failsafe - ensure we reset the transition flag in case of panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("CRITICAL ERROR in map transition: %v\n", r)
			GetDebugLog().Add(fmt.Sprintf("CRITICAL ERROR in map transition: %v", r))
			s.transitionInProgress = false // Reset flag
		}
	}()

	// Get current map type
	activeMap := s.GetActiveMap()
	if activeMap == nil {
		fmt.Println("ERROR: No active map found during transition")
		GetDebugLog().Add("ERROR: No active map found during transition")
		s.transitionInProgress = false // Reset flag
		return
	}

	// Debug current state before transition
	var activeMapType string
	var activeMapLevel int
	if typeComp, exists := world.GetComponent(activeMap.ID, components.MapType); exists {
		mapTypeComp := typeComp.(*components.MapTypeComponent)
		activeMapType = mapTypeComp.MapType
		activeMapLevel = mapTypeComp.Level
	}
	fmt.Printf("TRANSITION DEBUG: Current active map: ID=%d Type=%s Level=%d\n",
		activeMap.ID, activeMapType, activeMapLevel)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Current active map: ID=%d Type=%s Level=%d",
		activeMap.ID, activeMapType, activeMapLevel))

	// Get player map context for debugging
	playerEntity := s.getPlayer()
	if playerEntity == nil {
		fmt.Println("ERROR: Player entity not found during transition debug")
		GetDebugLog().Add("ERROR: Player entity not found during transition debug")
		s.transitionInProgress = false
		return
	}

	var playerMapID ecs.EntityID
	if contextComp, exists := world.GetComponent(playerEntity.ID, components.MapContextID); exists {
		mapContext := contextComp.(*components.MapContextComponent)
		playerMapID = mapContext.MapID
		fmt.Printf("TRANSITION DEBUG: Player map context before transition: %d\n", playerMapID)
		GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Player map context before transition: %d", playerMapID))
	} else {
		fmt.Println("ERROR: Player has no map context component before transition")
		GetDebugLog().Add("ERROR: Player has no map context component before transition")
	}

	mapTypeCompInterface, exists := world.GetComponent(activeMap.ID, components.MapType)
	if !exists {
		fmt.Println("ERROR: Map type component not found during transition")
		GetDebugLog().Add("ERROR: Map type component not found during transition")
		s.transitionInProgress = false // Reset flag
		return
	}
	currentMapType := mapTypeCompInterface.(*components.MapTypeComponent)

	// Determine target map type based on current map and stair direction
	var targetMapType *components.MapTypeComponent
	var targetMap *ecs.Entity

	// Log starting transition attempt
	fmt.Printf("TRANSITION START: From %s (Level %d, ID: %d)\n",
		currentMapType.MapType, currentMapType.Level, activeMap.ID)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION START: From %s (Level %d, ID: %d)",
		currentMapType.MapType, currentMapType.Level, activeMap.ID))

	if currentMapType.MapType == "worldmap" && tileType == components.TileStairsDown {
		// Going down from world map to a dungeon
		fmt.Println("TRANSITION: Going from worldmap to dungeon")
		targetMapType = components.NewMapTypeComponent("dungeon", 1) // Level 1 dungeon
		targetMap = s.GetMapByType("dungeon", 1)

		// If no dungeon exists, this is an error since all maps should be created during init
		if targetMap == nil {
			fmt.Println("ERROR: Attempted to transition to non-existent dungeon")
			GetDebugLog().Add("ERROR: Attempted to transition to non-existent dungeon")
			s.transitionInProgress = false
			return
		}

		fmt.Printf("TRANSITION TARGET: Going from worldmap to dungeon (ID: %d)\n", targetMap.ID)
		GetDebugLog().Add(fmt.Sprintf("TRANSITION TARGET: Going from worldmap to dungeon (ID: %d)", targetMap.ID))
	} else if currentMapType.MapType == "dungeon" && tileType == components.TileStairsUp {
		// Going up from dungeon to world map
		fmt.Println("TRANSITION: Going from dungeon to worldmap")
		targetMapType = components.NewMapTypeComponent("worldmap", 0)
		targetMap = s.GetMapByType("worldmap", 0)

		// If no world map exists, this is an error since all maps should be created during init
		if targetMap == nil {
			fmt.Println("ERROR: Attempted to transition to non-existent world map")
			GetDebugLog().Add("ERROR: Attempted to transition to non-existent world map")
			s.transitionInProgress = false
			return
		}

		fmt.Printf("TRANSITION TARGET: Going from dungeon to worldmap (ID: %d)\n", targetMap.ID)
		GetDebugLog().Add(fmt.Sprintf("TRANSITION TARGET: Going from dungeon to worldmap (ID: %d)", targetMap.ID))
	} else if currentMapType.MapType == "dungeon" && tileType == components.TileStairsDown {
		// Going down to a deeper dungeon level
		nextLevel := currentMapType.Level + 1
		fmt.Printf("TRANSITION: Going from dungeon level %d to level %d\n", currentMapType.Level, nextLevel)
		targetMapType = components.NewMapTypeComponent("dungeon", nextLevel)
		targetMap = s.GetMapByType("dungeon", nextLevel)

		// If no dungeon at this level exists, this is an error since all maps should be created during init
		if targetMap == nil {
			fmt.Printf("ERROR: Attempted to transition to non-existent dungeon level %d\n", nextLevel)
			GetDebugLog().Add(fmt.Sprintf("ERROR: Attempted to transition to non-existent dungeon level %d", nextLevel))
			s.transitionInProgress = false
			return
		}

		fmt.Printf("TRANSITION TARGET: Going from dungeon level %d to level %d (ID: %d)\n",
			currentMapType.Level, nextLevel, targetMap.ID)
		GetDebugLog().Add(fmt.Sprintf("TRANSITION TARGET: Going from dungeon level %d to level %d (ID: %d)",
			currentMapType.Level, nextLevel, targetMap.ID))
	} else {
		// Invalid transition
		fmt.Printf("TRANSITION CANCELLED: Invalid transition (map type: %s, tile type: %d)\n",
			currentMapType.MapType, tileType)
		GetMessageLog().Add("You can't go that way.")
		GetDebugLog().Add("TRANSITION CANCELLED: Invalid transition")
		s.transitionInProgress = false // Reset flag
		return
	}

	// Log target map type for debugging
	if targetMap != nil {
		var targetMapTypeStr string
		var targetMapLevel int
		if typeComp, exists := world.GetComponent(targetMap.ID, components.MapType); exists {
			mapTypeComp := typeComp.(*components.MapTypeComponent)
			targetMapTypeStr = mapTypeComp.MapType
			targetMapLevel = mapTypeComp.Level
			fmt.Printf("TRANSITION DEBUG: Target map details: ID=%d Type=%s Level=%d\n",
				targetMap.ID, targetMapTypeStr, targetMapLevel)
			GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Target map details: ID=%d Type=%s Level=%d",
				targetMap.ID, targetMapTypeStr, targetMapLevel))
		} else {
			fmt.Println("ERROR: Target map has no map type component")
			GetDebugLog().Add("ERROR: Target map has no map type component")
		}
	} else {
		fmt.Println("ERROR: Target map is nil after selection")
		GetDebugLog().Add("ERROR: Target map is nil after selection")
		s.transitionInProgress = false
		return
	}

	// Check that we have a valid target map before proceeding
	if targetMap == nil || targetMap.ID == 0 {
		fmt.Println("ERROR: Target map is invalid")
		GetDebugLog().Add("ERROR: Target map is invalid")
		s.transitionInProgress = false
		return
	}

	// Position the player appropriately on the new map
	var targetX, targetY int
	if tileType == components.TileStairsDown {
		// Find stairs up on the target map
		targetMapComp, exists := world.GetComponent(targetMap.ID, components.MapComponentID)
		if !exists {
			fmt.Println("ERROR: Target map has no map component")
			GetDebugLog().Add("ERROR: Target map has no map component")
			s.transitionInProgress = false // Reset flag
			return
		}

		foundStairs := false
		tmc := targetMapComp.(*components.MapComponent)
		for y := 0; y < tmc.Height; y++ {
			for x := 0; x < tmc.Width; x++ {
				if tmc.Tiles[y][x] == components.TileStairsUp {
					targetX, targetY = x, y
					foundStairs = true
					break
				}
			}
			if foundStairs {
				break
			}
		}

		if !foundStairs {
			// If no stairs found, place player at an empty spot
			fmt.Println("Could not find stairs up on target map, finding empty position")
			mapSystem := s.getMapSystem()
			if mapSystem != nil {
				targetX, targetY = mapSystem.FindEmptyPosition(tmc)
			} else {
				// Simple fallback
				targetX, targetY = tmc.Width/2, tmc.Height/2
			}
		}
	} else if tileType == components.TileStairsUp {
		// Player is heading to the world map or previous level
		if targetMapType.MapType == "worldmap" {
			// For transitions to the world map
			fmt.Println("Transitioning to world map")
			GetDebugLog().Add("Transitioning to world map")

			// For world map, always position at center (railway station) at 100,100
			targetX, targetY = 100, 100
			fmt.Printf("TRANSITION: Positioning player at railway station (100,100) on world map\n")
			GetDebugLog().Add("TRANSITION: Positioning player at railway station (100,100) on world map")
		} else {
			// For dungeon-to-dungeon transitions (going back up a level)
			// If we have a last position, use it
			if s.lastPosition != nil && s.lastMapID == targetMap.ID {
				targetX, targetY = s.lastPosition.X, s.lastPosition.Y
			} else {
				// If no last position, find an empty spot
				fmt.Println("No last position for dungeon-to-dungeon transition, finding empty spot")
				targetMapComp, exists := world.GetComponent(targetMap.ID, components.MapComponentID)
				if !exists {
					s.transitionInProgress = false // Reset flag
					return
				}

				mapSystem := s.getMapSystem()
				if mapSystem != nil {
					targetX, targetY = mapSystem.FindEmptyPosition(targetMapComp.(*components.MapComponent))
				} else {
					// Simple fallback
					tmc := targetMapComp.(*components.MapComponent)
					targetX, targetY = tmc.Width/2, tmc.Height/2
				}
			}
		}
	}

	fmt.Printf("TRANSITION: Player target position on new map: %d,%d\n", targetX, targetY)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION: Player target position on new map: %d,%d", targetX, targetY))

	// CRITICAL ORDER OF OPERATIONS:
	// 1. First UPDATE THE ACTIVE MAP
	GetDebugLog().Add("=============================================")
	GetDebugLog().Add("TRANSITION STEP 1: Setting active map")
	// Get pre-update state for logging
	var oldActiveMapID = s.activeMapID
	s.SetActiveMap(targetMap)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Changed active map from %d to %d", oldActiveMapID, s.activeMapID))
	if s.activeMapID != targetMap.ID {
		GetDebugLog().Add("ERROR: Active map was not updated correctly! This is a critical error.")
	}

	// 2. Then update player's map context to match the new active map
	GetDebugLog().Add("TRANSITION STEP 2: Updating player's map context")
	var oldPlayerMapContext ecs.EntityID = 0
	if world.HasComponent(playerEntity.ID, components.MapContextID) {
		mapContextComp, _ := world.GetComponent(playerEntity.ID, components.MapContextID)
		oldPlayerMapContext = mapContextComp.(*components.MapContextComponent).MapID
		mapContextComp.(*components.MapContextComponent).MapID = targetMap.ID
		GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Updated player entity %d map context from %d to %d",
			playerEntity.ID, oldPlayerMapContext, targetMap.ID))

		// Verify the update
		if mapContextComp.(*components.MapContextComponent).MapID != targetMap.ID {
			GetDebugLog().Add("ERROR: Player map context was not updated correctly!")
		}
	} else {
		world.AddComponent(playerEntity.ID, components.MapContextID, components.NewMapContextComponent(targetMap.ID))
		GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Added new map context to player: %d", targetMap.ID))

		// Verify the component was added
		if !world.HasComponent(playerEntity.ID, components.MapContextID) {
			GetDebugLog().Add("ERROR: Failed to add map context to player!")
		}
	}

	// 3. Update player position
	GetDebugLog().Add("TRANSITION STEP 3: Updating player position")
	var oldX, oldY = playerPos.X, playerPos.Y
	playerPos.X = targetX
	playerPos.Y = targetY
	GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Updated player position from (%d,%d) to (%d,%d)",
		oldX, oldY, playerPos.X, playerPos.Y))

	// Verify update
	if playerPos.X != targetX || playerPos.Y != targetY {
		GetDebugLog().Add("ERROR: Player position was not updated correctly!")
	}

	// 4. Force camera update after map change
	GetDebugLog().Add("TRANSITION STEP 4: Updating camera position")
	s.updateCameraPosition(world, targetX, targetY)

	// Log the transition completion
	if targetMapType.MapType == "worldmap" {
		fmt.Println("TRANSITION COMPLETE: Player now on world map")
		GetMessageLog().Add("You climb the stairs and emerge onto the surface.")
		GetDebugLog().Add("TRANSITION COMPLETE: Player now on world map")
	} else {
		if currentMapType.MapType == "worldmap" {
			fmt.Println("TRANSITION COMPLETE: Player now in dungeon")
			GetMessageLog().Add("You descend into the darkness below.")
			GetDebugLog().Add("TRANSITION COMPLETE: Player now in dungeon")
		} else if currentMapType.Level < targetMapType.Level {
			fmt.Printf("TRANSITION COMPLETE: Player now in dungeon level %d\n", targetMapType.Level)
			GetMessageLog().Add("You descend deeper into the dungeon.")
			GetDebugLog().Add(fmt.Sprintf("TRANSITION COMPLETE: Player now in dungeon level %d", targetMapType.Level))
		} else {
			fmt.Printf("TRANSITION COMPLETE: Player now in dungeon level %d\n", targetMapType.Level)
			GetMessageLog().Add("You climb back to the previous level.")
			GetDebugLog().Add(fmt.Sprintf("TRANSITION COMPLETE: Player now in dungeon level %d", targetMapType.Level))
		}
	}

	// Reset the AI pathfinding system's turn processed flag to avoid AI processing in the new map
	for _, system := range world.GetSystems() {
		if aiPathfinding, ok := system.(*AIPathfindingSystem); ok {
			aiPathfinding.ResetTurn()
			GetDebugLog().Add("TRANSITION: Reset AI pathfinding turn flag to prevent immediate AI turn")
			break
		}
	}

	// Verify the transition completed correctly
	GetDebugLog().Add("TRANSITION VERIFICATION:")

	// Check player map context
	if world.HasComponent(playerEntity.ID, components.MapContextID) {
		mapContextComp, _ := world.GetComponent(playerEntity.ID, components.MapContextID)
		playerMapID := mapContextComp.(*components.MapContextComponent).MapID
		GetDebugLog().Add(fmt.Sprintf("- Player map context ID: %d", playerMapID))

		if playerMapID != targetMap.ID {
			GetDebugLog().Add("  ERROR: Player map context doesn't match target map!")
		} else {
			GetDebugLog().Add("  OK: Player map context matches target map")
		}
	} else {
		GetDebugLog().Add("  ERROR: Player has no map context component after transition!")
	}

	// Check active map
	GetDebugLog().Add(fmt.Sprintf("- Active map ID: %d", s.activeMapID))
	if s.activeMapID != targetMap.ID {
		GetDebugLog().Add("  ERROR: Active map doesn't match target map!")
	} else {
		GetDebugLog().Add("  OK: Active map matches target map")
	}

	// Check player position
	GetDebugLog().Add(fmt.Sprintf("- Player position: (%d,%d)", playerPos.X, playerPos.Y))
	if playerPos.X != targetX || playerPos.Y != targetY {
		GetDebugLog().Add("  ERROR: Player position doesn't match target position!")
	} else {
		GetDebugLog().Add("  OK: Player position matches target position")
	}

	GetDebugLog().Add("=====================================")
	GetDebugLog().Add("TRANSITION: COMPLETED MAP TRANSITION")
	GetDebugLog().Add("=====================================")

	// Reset the transition flag now that we're done
	s.transitionInProgress = false
}

// updateCameraPosition centers the camera on the given position
func (s *MapRegistrySystem) updateCameraPosition(world *ecs.World, x, y int) {
	// Find camera entity
	cameraEntities := world.GetEntitiesWithComponent(components.Camera)
	if len(cameraEntities) == 0 {
		return
	}

	// Get active map dimensions for boundary checks
	activeMap := s.GetActiveMap()
	if activeMap == nil {
		return
	}

	mapComp, exists := world.GetComponent(activeMap.ID, components.MapComponentID)
	if !exists {
		return
	}
	mc := mapComp.(*components.MapComponent)

	// Update camera position - directly set it rather than relying on player position
	for _, entity := range cameraEntities {
		cameraComp, exists := world.GetComponent(entity.ID, components.Camera)
		if !exists {
			continue
		}

		camera := cameraComp.(*components.CameraComponent)
		// Set camera position to center on coordinates
		camera.X = x - config.GameScreenWidth/2
		camera.Y = y - config.GameScreenHeight/2

		// Constrain camera to map boundaries
		if camera.X < 0 {
			camera.X = 0
		}
		if camera.Y < 0 {
			camera.Y = 0
		}
		if camera.X > mc.Width-config.GameScreenWidth {
			camera.X = mc.Width - config.GameScreenWidth
		}
		if camera.Y > mc.Height-config.GameScreenHeight {
			camera.Y = mc.Height - config.GameScreenHeight
		}

		// Log camera position update
		GetDebugLog().Add(fmt.Sprintf("Camera position updated to (%d,%d) for map %d",
			camera.X, camera.Y, activeMap.ID))
	}
}

// IsTransitionInProgress returns whether a map transition is currently in progress
func (s *MapRegistrySystem) IsTransitionInProgress() bool {
	return s.transitionInProgress
}
