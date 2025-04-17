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
	// Store world reference
	s.world = world

	// Subscribe to examine events
	world.GetEventManager().Subscribe(EventExamine, func(event ecs.Event) {
		examineEvent := event.(ExamineEvent)
		s.HandleEvent(world, examineEvent)
	})
}

// Update is called every frame but only processes transitions during player turns
func (s *MapRegistrySystem) Update(world *ecs.World, dt float64) {
	// No processing needed every frame in a turn-based game
}

// HandleEvent processes map transition events
func (s *MapRegistrySystem) HandleEvent(world *ecs.World, event ecs.Event) {
	switch e := event.(type) {
	case ExamineEvent:
		// Check if the examined entity is stairs
		entity := s.world.GetEntity(e.TargetID)
		if entity != nil && entity.HasTag("stairs") {
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

			// Get stairs position
			stairsPos, exists := s.world.GetComponent(entity.ID, components.Position)
			if !exists {
				return
			}
			stPos := stairsPos.(*components.PositionComponent)

			// Check if player is on the stairs
			if pos.X == stPos.X && pos.Y == stPos.Y {
				GetDebugLog().Add(fmt.Sprintf("Player examining stairs at (%d,%d)", stPos.X, stPos.Y))
				s.handleMapTransitions(world)
			} else {
				GetMessageLog().AddEnvironment("You need to be on the stairs to use them.")
			}
		}
	}
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

	// Get current map component
	mapCompInterface, exists := world.GetComponent(activeMap.ID, components.MapComponentID)
	if !exists {
		fmt.Println("ERROR: Current map has no map component")
		GetDebugLog().Add("ERROR: Current map has no map component")
		s.transitionInProgress = false // Reset flag
		return
	}
	currentMap := mapCompInterface.(*components.MapComponent)

	// Get transition data for current position
	transitionData, hasTransition := currentMap.GetTransition(playerPos.X, playerPos.Y)
	if !hasTransition {
		fmt.Println("ERROR: No transition data found at current position")
		GetDebugLog().Add("ERROR: No transition data found at current position")
		s.transitionInProgress = false // Reset flag
		return
	}

	// Get target map
	targetMap := world.GetEntity(transitionData.TargetMapID)
	if targetMap == nil {
		fmt.Println("ERROR: Target map entity not found")
		GetDebugLog().Add("ERROR: Target map entity not found")
		s.transitionInProgress = false // Reset flag
		return
	}

	// Get target map type for logging
	var targetMapType string
	var targetMapLevel int
	if typeComp, exists := world.GetComponent(targetMap.ID, components.MapType); exists {
		mapTypeComp := typeComp.(*components.MapTypeComponent)
		targetMapType = mapTypeComp.MapType
		targetMapLevel = mapTypeComp.Level
	}

	// Log transition details
	fmt.Printf("TRANSITION: From map %d to map %d (%s level %d)\n",
		activeMap.ID, targetMap.ID, targetMapType, targetMapLevel)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION: From map %d to map %d (%s level %d)",
		activeMap.ID, targetMap.ID, targetMapType, targetMapLevel))

	// Get player entity
	playerEntity := s.getPlayer()
	if playerEntity == nil {
		fmt.Println("ERROR: Player entity not found")
		GetDebugLog().Add("ERROR: Player entity not found")
		s.transitionInProgress = false
		return
	}

	// CRITICAL ORDER OF OPERATIONS:
	// 1. First UPDATE THE ACTIVE MAP
	GetDebugLog().Add("=============================================")
	GetDebugLog().Add("TRANSITION STEP 1: Setting active map")
	var oldActiveMapID = s.activeMapID
	s.SetActiveMap(targetMap)
	GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Changed active map from %d to %d", oldActiveMapID, s.activeMapID))

	// 2. Then update player's map context to match the new active map
	GetDebugLog().Add("TRANSITION STEP 2: Updating player's map context")
	var oldPlayerMapContext ecs.EntityID = 0
	if world.HasComponent(playerEntity.ID, components.MapContextID) {
		mapContextComp, _ := world.GetComponent(playerEntity.ID, components.MapContextID)
		oldPlayerMapContext = mapContextComp.(*components.MapContextComponent).MapID
		mapContextComp.(*components.MapContextComponent).MapID = targetMap.ID
		GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Updated player entity %d map context from %d to %d",
			playerEntity.ID, oldPlayerMapContext, targetMap.ID))
	} else {
		world.AddComponent(playerEntity.ID, components.MapContextID, components.NewMapContextComponent(targetMap.ID))
		GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Added new map context to player: %d", targetMap.ID))
	}

	// 3. Update player position using transition data
	GetDebugLog().Add("TRANSITION STEP 3: Updating player position")
	var oldX, oldY = playerPos.X, playerPos.Y
	playerPos.X = transitionData.TargetX
	playerPos.Y = transitionData.TargetY
	GetDebugLog().Add(fmt.Sprintf("TRANSITION DEBUG: Updated player position from (%d,%d) to (%d,%d)",
		oldX, oldY, playerPos.X, playerPos.Y))

	// 4. Force camera update after map change
	GetDebugLog().Add("TRANSITION STEP 4: Updating camera position")
	s.updateCameraPosition(world, playerPos.X, playerPos.Y)

	// Log the transition completion
	if targetMapType == "worldmap" {
		fmt.Println("TRANSITION COMPLETE: Player now on world map")
		GetMessageLog().Add("You climb the stairs and emerge onto the surface.")
		GetDebugLog().Add("TRANSITION COMPLETE: Player now on world map")
	} else {
		fmt.Printf("TRANSITION COMPLETE: Player now in dungeon level %d\n", targetMapLevel)
		GetMessageLog().Add(fmt.Sprintf("You %s to level %d.",
			map[bool]string{true: "descend", false: "climb"}[tileType == components.TileStairsDown],
			targetMapLevel))
		GetDebugLog().Add(fmt.Sprintf("TRANSITION COMPLETE: Player now in dungeon level %d", targetMapLevel))
	}

	// Reset the AI pathfinding system's turn processed flag
	for _, system := range world.GetSystems() {
		if aiPathfinding, ok := system.(*AIPathfindingSystem); ok {
			aiPathfinding.ResetTurn()
			GetDebugLog().Add("TRANSITION: Reset AI pathfinding turn flag")
			break
		}
	}

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

// Clear resets the map registry state
func (s *MapRegistrySystem) Clear() {
	// Create a new empty map to replace the old one
	s.maps = make(map[string][]*ecs.Entity)
	s.activeMapID = 0
	s.lastMapID = 0
	s.lastPosition = nil
	s.transitionInProgress = false
}
