package systems

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// AIPathEvent is emitted when AI pathfinding completes
type AIPathEvent struct {
	EntityID ecs.EntityID          // The AI entity
	Path     []components.PathNode // The calculated path
	TargetX  int                   // Target X coordinate
	TargetY  int                   // Target Y coordinate
	Visible  bool                  // Whether the target is visible
}

// Type returns the event type
func (e AIPathEvent) Type() ecs.EventType {
	return "ai_path_event" // Define a constant for this elsewhere if desired
}

// AIPathfindingSystem handles AI vision and path calculation
type AIPathfindingSystem struct {
	turnProcessed bool // Flag to track if AI paths have been processed this game turn
}

// NewAIPathfindingSystem creates a new AI pathfinding system
func NewAIPathfindingSystem() *AIPathfindingSystem {
	return &AIPathfindingSystem{
		turnProcessed: false,
	}
}

// Initialize sets up event listeners for the AI system
func (s *AIPathfindingSystem) Initialize(world *ecs.World) {
	// Subscribe to player movement events
	world.GetEventManager().Subscribe(EventMovement, func(event ecs.Event) {
		s.HandleEvent(world, event)
	})

	// Subscribe to rest events as well - AI should also act when player rests
	world.GetEventManager().Subscribe(EventRest, func(event ecs.Event) {
		s.HandleEvent(world, event)
	})
}

// HandleEvent processes events that the AI system is interested in
func (s *AIPathfindingSystem) HandleEvent(world *ecs.World, event ecs.Event) {
	// Check if this is a player movement event
	if moveEvent, ok := event.(PlayerMoveEvent); ok {
		// Player has moved, so reset the turn flag to allow AI processing
		s.turnProcessed = false

		// Log for debugging using message system
		GetMessageLog().Add("DEBUG: Player moved to: " + strconv.Itoa(moveEvent.ToX) + "," + strconv.Itoa(moveEvent.ToY) + " AI can now recalculate paths")
	} else if entityMove, ok := event.(EntityMoveEvent); ok {
		// Check if this is the player moving (not an AI entity)
		playerEntities := world.GetEntitiesWithTag("player")
		if len(playerEntities) > 0 && ecs.EntityID(playerEntities[0].ID) == entityMove.EntityID {
			s.turnProcessed = false
			// Log for debugging using message system
			GetMessageLog().Add("DEBUG: Player entity moved, AI can now recalculate paths")
		}
	} else if _, ok := event.(RestEvent); ok {
		// Player has rested, so reset the turn flag to allow AI processing
		s.turnProcessed = false

		// Log for debugging
		GetDebugLog().Add("DEBUG: Player rested, AI will now take its turn")
	}
}

// Update processes AI behavior for entities with AI components
func (s *AIPathfindingSystem) Update(world *ecs.World, dt float64) {
	// Do nothing if we've already processed AI this turn
	if s.turnProcessed {
		return
	}

	// Get the player entity for reference
	playerEntities := world.GetEntitiesWithTag("player")
	if len(playerEntities) == 0 {
		return
	}

	playerID := playerEntities[0].ID
	var playerPos *components.PositionComponent
	if comp, exists := world.GetComponent(playerID, components.Position); exists {
		playerPos = comp.(*components.PositionComponent)
	} else {
		return
	}

	// Get the active map ID from MapRegistrySystem (preferred)
	var activeMapID ecs.EntityID
	for _, system := range world.GetSystems() {
		if mapReg, ok := system.(interface{ GetActiveMap() *ecs.Entity }); ok {
			if activeMap := mapReg.GetActiveMap(); activeMap != nil {
				activeMapID = activeMap.ID
				break
			}
		}
	}

	// If we couldn't get an active map ID, we can't proceed
	if activeMapID == 0 {
		return
	}

	// Get map for pathfinding
	mapComp, exists := world.GetComponent(activeMapID, components.MapComponentID)
	if !exists {
		return
	}
	gameMap := mapComp.(*components.MapComponent)

	// Process all entities with AI components
	aiEntities := world.GetEntitiesWithTag("ai")
	for _, entity := range aiEntities {
		// Skip entities that aren't on the active map
		if world.HasComponent(entity.ID, components.MapContextID) {
			mapContextComp, _ := world.GetComponent(entity.ID, components.MapContextID)
			mapContext := mapContextComp.(*components.MapContextComponent)

			// Skip if not on the active map
			if mapContext.MapID != activeMapID {
				continue
			}
		} else {
			// Skip entities without a map context
			continue
		}

		aiComp, _ := world.GetComponent(entity.ID, components.AI)
		ai := aiComp.(*components.AIComponent)

		// Get the entity's position
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		if !hasPos {
			continue
		}
		pos := posComp.(*components.PositionComponent)

		// Process AI based on type
		switch ai.Type {
		case "slow_chase", "slow_wander":
			s.processPathfinding(world, entity.ID, ai, pos, playerPos, gameMap)
			// Add other AI types here as needed
		}
	}

	// Mark turn as processed
	s.turnProcessed = true
}

// processPathfinding handles pathfinding for AI entities
func (s *AIPathfindingSystem) processPathfinding(world *ecs.World, entityID ecs.EntityID, ai *components.AIComponent, pos *components.PositionComponent, playerPos *components.PositionComponent, gameMap *components.MapComponent) {
	// Check if player is in sight
	playerVisible := s.canSee(pos.X, pos.Y, playerPos.X, playerPos.Y, ai.SightRange, gameMap)
	// GetMessageLog().Add(fmt.Sprintf("DEBUG: AI at %d,%d checking for player at %d,%d (visible: %v)", pos.X, pos.Y, playerPos.X, playerPos.Y, playerVisible))

	var targetX, targetY int
	var path []components.PathNode

	if playerVisible {
		// Player is visible, update last known position
		ai.LastKnownTargetX = playerPos.X
		ai.LastKnownTargetY = playerPos.Y
		targetX, targetY = playerPos.X, playerPos.Y
		// GetMessageLog().Add(fmt.Sprintf("DEBUG: Updated target pos to %d,%d", playerPos.X, playerPos.Y))

		// Calculate path to player
		path = s.findPath(pos.X, pos.Y, targetX, targetY, gameMap)
	} else if ai.Type == "slow_wander" {
		// For slow_wander AI, generate a random direction when player not visible
		directions := []struct{ dx, dy int }{
			{1, 0},  // Right
			{-1, 0}, // Left
			{0, 1},  // Down
			{0, -1}, // Up
		}

		// Try to find a valid random direction
		validMoves := []components.PathNode{}
		for _, dir := range directions {
			newX, newY := pos.X+dir.dx, pos.Y+dir.dy
			if s.isValidMove(world, newX, newY, gameMap) {
				validMoves = append(validMoves, components.PathNode{X: newX, Y: newY})
			}
		}

		// If there are valid moves, choose one randomly
		if len(validMoves) > 0 {
			randomMove := validMoves[rand.Intn(len(validMoves))]
			path = []components.PathNode{randomMove}
			targetX, targetY = randomMove.X, randomMove.Y
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AI wandering to random direction: %d,%d", targetX, targetY))
		} else {
			// No valid moves, just stay in place
			path = []components.PathNode{}
		}
	} else if ai.LastKnownTargetX != 0 || ai.LastKnownTargetY != 0 {
		// Use last known player position (for slow_chase and default behavior)
		targetX, targetY = ai.LastKnownTargetX, ai.LastKnownTargetY
		path = s.findPath(pos.X, pos.Y, targetX, targetY, gameMap)
	} else {
		// No target
		return
	}

	// Store path in AI component for reference
	ai.Path = path

	// Emit path event for the turn processor to handle
	world.EmitEvent(AIPathEvent{
		EntityID: entityID,
		Path:     path,
		TargetX:  targetX,
		TargetY:  targetY,
		Visible:  playerVisible,
	})

	// GetMessageLog().Add(fmt.Sprintf("DEBUG: AI path calculated, length: %d", len(path)))
}

// canSee checks if there's a clear line of sight between two points
func (s *AIPathfindingSystem) canSee(x1, y1, x2, y2, sightRange int, gameMap *components.MapComponent) bool {
	// First check range
	distance := int(math.Sqrt(float64((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1))))
	if distance > sightRange {
		return false
	}

	// If in range, check line of sight using Bresenham's line algorithm
	points := s.getLinePoints(x1, y1, x2, y2)
	for _, point := range points {
		// Skip the starting point
		if point.X == x1 && point.Y == y1 {
			continue
		}
		// If we hit a wall, line of sight is blocked
		if gameMap.IsWall(point.X, point.Y) {
			return false
		}
	}

	return true
}

// getLinePoints returns all points on a line between (x1,y1) and (x2,y2)
func (s *AIPathfindingSystem) getLinePoints(x1, y1, x2, y2 int) []components.PathNode {
	points := []components.PathNode{}

	dx := int(math.Abs(float64(x2 - x1)))
	dy := int(math.Abs(float64(y2 - y1)))
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		points = append(points, components.PathNode{X: x1, Y: y1})
		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}

	return points
}

// isValidMove checks if a position is a valid movement destination
func (s *AIPathfindingSystem) isValidMove(world *ecs.World, x, y int, gameMap *components.MapComponent) bool {
	// Check for walls
	if gameMap.IsWall(x, y) {
		return false
	}

	// Check for entity collision
	for _, entity := range world.GetAllEntities() {
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		if !hasPos {
			continue
		}

		pos := posComp.(*components.PositionComponent)
		if pos.X == x && pos.Y == y {
			// Position is occupied by an entity, check if it blocks
			if collComp, hasCol := world.GetComponent(entity.ID, components.Collision); hasCol {
				collision := collComp.(*components.CollisionComponent)
				if collision.Blocks {
					return false
				}
			}
		}
	}

	return true
}

// findPath uses A* pathfinding to find a path between two points
func (s *AIPathfindingSystem) findPath(startX, startY, targetX, targetY int, gameMap *components.MapComponent) []components.PathNode {
	// A* Pathfinding implementation
	openSet := make(PriorityQueue, 0)
	heap.Init(&openSet)

	// Maps for tracking
	cameFrom := make(map[Point]Point)
	gScore := make(map[Point]int)
	fScore := make(map[Point]int)
	inOpenSet := make(map[Point]bool)

	start := Point{X: startX, Y: startY}
	goal := Point{X: targetX, Y: targetY}

	// Initialize starting node
	gScore[start] = 0
	fScore[start] = s.heuristic(start, goal)
	startItem := &Item{
		value:    start,
		priority: fScore[start],
		index:    0,
	}
	heap.Push(&openSet, startItem)
	inOpenSet[start] = true

	// Main A* loop
	for openSet.Len() > 0 {
		current := heap.Pop(&openSet).(*Item).value.(Point)
		inOpenSet[current] = false

		if current == goal {
			// Path found, reconstruct and return it
			return s.reconstructPath(cameFrom, current)
		}

		// Check neighbors (4-directional movement)
		neighbors := []Point{
			{X: current.X + 1, Y: current.Y},
			{X: current.X - 1, Y: current.Y},
			{X: current.X, Y: current.Y + 1},
			{X: current.X, Y: current.Y - 1},
		}

		for _, neighbor := range neighbors {
			// Skip if out of bounds or wall
			if neighbor.X < 0 || neighbor.X >= gameMap.Width ||
				neighbor.Y < 0 || neighbor.Y >= gameMap.Height ||
				gameMap.IsWall(neighbor.X, neighbor.Y) {
				continue
			}

			// Calculate score
			tentativeGScore := gScore[current] + 1 // Cost is always 1 for adjacent cells

			_, neighborExists := gScore[neighbor]
			if !neighborExists {
				gScore[neighbor] = math.MaxInt32
			}

			if tentativeGScore < gScore[neighbor] {
				// This is a better path
				cameFrom[neighbor] = current
				gScore[neighbor] = tentativeGScore
				fScore[neighbor] = gScore[neighbor] + s.heuristic(neighbor, goal)

				if !inOpenSet[neighbor] {
					newItem := &Item{
						value:    neighbor,
						priority: fScore[neighbor],
					}
					heap.Push(&openSet, newItem)
					inOpenSet[neighbor] = true
				}
			}
		}
	}

	// No path found
	return []components.PathNode{}
}

// reconstructPath builds the path from start to goal
func (s *AIPathfindingSystem) reconstructPath(cameFrom map[Point]Point, current Point) []components.PathNode {
	path := []components.PathNode{}

	for {
		path = append([]components.PathNode{{X: current.X, Y: current.Y}}, path...)
		next, exists := cameFrom[current]
		if !exists {
			break
		}
		current = next
	}

	// Remove the first node which is the starting position
	if len(path) > 0 {
		path = path[1:]
	}

	return path
}

// heuristic estimates the cost to reach the goal
func (s *AIPathfindingSystem) heuristic(a, b Point) int {
	// Manhattan distance
	return int(math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)))
}

// Point represents a 2D point with integer coordinates
type Point struct {
	X, Y int
}

// ResetTurn resets the turn processed flag to allow AI processing in the next turn
func (s *AIPathfindingSystem) ResetTurn() {
	s.turnProcessed = false
}

// PriorityQueue implementation for A* pathfinding
type Item struct {
	value    interface{}
	priority int
	index    int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
