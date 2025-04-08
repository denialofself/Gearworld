package systems

import (
	"container/heap"
	"fmt"
	"math"
	"strconv"

	"ebiten-rogue/components"
	"ebiten-rogue/ecs"
)

// AISystem handles AI behavior for entities
type AISystem struct {
	turnProcessed bool // Flag to track if AI turns have been processed this game turn
}

// NewAISystem creates a new AI system
func NewAISystem() *AISystem {
	return &AISystem{
		turnProcessed: false,
	}
}

// Initialize sets up event listeners for the AI system
func (s *AISystem) Initialize(world *ecs.World) {
	// Subscribe to player movement events
	world.GetEventManager().Subscribe(EventMovement, func(event ecs.Event) {
		s.HandleEvent(world, event)
	})
}

// HandleEvent processes events that the AI system is interested in
func (s *AISystem) HandleEvent(world *ecs.World, event ecs.Event) {
	// Check if this is a player movement event
	if moveEvent, ok := event.(PlayerMoveEvent); ok {
		// Player has moved, so reset the turn flag to allow AI processing
		s.turnProcessed = false

		// Log for debugging using message system
		GetMessageLog().Add("DEBUG: Player moved to: " + strconv.Itoa(moveEvent.ToX) + "," + strconv.Itoa(moveEvent.ToY) + " AI can now move")
	} else if entityMove, ok := event.(EntityMoveEvent); ok {
		// Check if this is the player moving (not an AI entity)
		playerEntities := world.GetEntitiesWithTag("player")
		if len(playerEntities) > 0 && ecs.EntityID(playerEntities[0].ID) == entityMove.EntityID {
			s.turnProcessed = false
			// Log for debugging using message system
			GetMessageLog().Add("DEBUG: Player entity moved, AI can now move")
		}
	}
}

// Update processes AI behavior for entities with AI components
func (s *AISystem) Update(world *ecs.World, dt float64) {
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

	// Get map for pathfinding
	mapEntities := world.GetEntitiesWithTag("map")
	if len(mapEntities) == 0 {
		return
	}

	mapComp, exists := world.GetComponent(mapEntities[0].ID, components.MapComponentID)
	if !exists {
		return
	}
	gameMap := mapComp.(*components.MapComponent)

	// Process all entities with AI components
	aiEntities := world.GetEntitiesWithTag("ai")
	for _, entity := range aiEntities {
		aiComp, _ := world.GetComponent(entity.ID, components.AI)
		ai := aiComp.(*components.AIComponent)

		// Skip AI entities with no action points
		if ai.ActionPoints <= 0 {
			// Rest and regain some action points
			ai.ActionPoints += 1
			if ai.ActionPoints > ai.MaxActionPoints {
				ai.ActionPoints = ai.MaxActionPoints
			}
			continue
		}

		// Get the entity's position
		posComp, hasPos := world.GetComponent(entity.ID, components.Position)
		if !hasPos {
			continue
		}
		pos := posComp.(*components.PositionComponent)

		// Process AI based on type
		switch ai.Type {
		case "slow_chase":
			s.handleSlowChase(world, uint64(entity.ID), ai, pos, playerPos, gameMap)
			// Add other AI types here as needed
		}
	}

	// Mark turn as processed
	s.turnProcessed = true
}

// handleSlowChase implements the slow_chase AI behavior
func (s *AISystem) handleSlowChase(world *ecs.World, entityID uint64, ai *components.AIComponent, pos *components.PositionComponent, playerPos *components.PositionComponent, gameMap *components.MapComponent) {
	// Cost for actions
	const (
		MoveCost = 2
		WaitCost = 0
	)

	// Debug info
	GetMessageLog().Add(fmt.Sprintf("DEBUG: AI at %d,%d checking for player at %d,%d AP:%d", pos.X, pos.Y, playerPos.X, playerPos.Y, ai.ActionPoints))

	// Check if player is in sight
	playerVisible := s.canSee(pos.X, pos.Y, playerPos.X, playerPos.Y, ai.SightRange, gameMap)
	GetMessageLog().Add(fmt.Sprintf("DEBUG: Player visible: %v Sight range: %d", playerVisible, ai.SightRange))

	if playerVisible {
		// Player is visible, update last known position
		ai.LastKnownTargetX = playerPos.X
		ai.LastKnownTargetY = playerPos.Y
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Updated target pos to %d,%d", playerPos.X, playerPos.Y))

		// Calculate path to player
		ai.Path = s.findPath(pos.X, pos.Y, playerPos.X, playerPos.Y, gameMap)
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Path length: %d", len(ai.Path)))
	}

	// If we have a path, follow it
	if len(ai.Path) > 0 {
		// Get the next step in the path
		nextStep := ai.Path[0]
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Next step: %d,%d", nextStep.X, nextStep.Y))

		// Check if we can move there
		canMove := s.isValidMove(world, nextStep.X, nextStep.Y, gameMap)
		GetMessageLog().Add(fmt.Sprintf("DEBUG: Can move: %v", canMove))

		if canMove && ai.ActionPoints >= MoveCost {
			// Move to the next step
			oldX, oldY := pos.X, pos.Y
			pos.X = nextStep.X
			pos.Y = nextStep.Y
			GetMessageLog().Add(fmt.Sprintf("DEBUG: Moving from %d,%d to %d,%d", oldX, oldY, pos.X, pos.Y))

			// Consume action points
			ai.ActionPoints -= MoveCost
			GetMessageLog().Add(fmt.Sprintf("DEBUG: AP remaining: %d", ai.ActionPoints))

			// Remove this step from the path
			ai.Path = ai.Path[1:]

			// Emit movement event
			world.EmitEvent(EntityMoveEvent{
				EntityID: ecs.EntityID(entityID),
				FromX:    oldX,
				FromY:    oldY,
				ToX:      pos.X,
				ToY:      pos.Y,
			})
			GetMessageLog().Add("DEBUG: Emitted movement event")
		} else if ai.ActionPoints >= WaitCost {
			// Can't move but can wait (might be blocked by another entity)
			ai.ActionPoints -= WaitCost
			GetMessageLog().Add(fmt.Sprintf("DEBUG: Can't move, waiting. AP: %d", ai.ActionPoints))
		} else {
			// Not enough action points, just wait
			statsComp, hasStats := world.GetComponent(ecs.EntityID(entityID), components.Stats)
			if hasStats {
				stats := statsComp.(*components.StatsComponent)
				ai.ActionPoints += stats.Recovery
				GetMessageLog().Add(fmt.Sprintf("DEBUG: Not enough AP, recovering %d points", stats.Recovery))
			} else {
				// Default recovery if no stats component
				ai.ActionPoints += 1
				GetMessageLog().Add("DEBUG: Not enough AP, recovering 1 point")
			}
			GetMessageLog().Add("DEBUG: Not enough AP, waiting")
		}
	} else if playerVisible {
		// Player is visible but no path
		// This could happen if player is surrounded or unreachable
		ai.ActionPoints -= WaitCost
		GetMessageLog().Add("DEBUG: Player visible but no path found")
	} else {
		// No path and no player visible, rest
		ai.ActionPoints = 0
		GetMessageLog().Add("DEBUG: No path and no player visible")
	}
}

// canSee checks if there's a clear line of sight between two points
func (s *AISystem) canSee(x1, y1, x2, y2, sightRange int, gameMap *components.MapComponent) bool {
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
func (s *AISystem) getLinePoints(x1, y1, x2, y2 int) []components.PathNode {
	points := []components.PathNode{}

	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
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
func (s *AISystem) isValidMove(world *ecs.World, x, y int, gameMap *components.MapComponent) bool {
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
func (s *AISystem) findPath(startX, startY, targetX, targetY int, gameMap *components.MapComponent) []components.PathNode {
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
func (s *AISystem) reconstructPath(cameFrom map[Point]Point, current Point) []components.PathNode {
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
func (s *AISystem) heuristic(a, b Point) int {
	// Manhattan distance
	return abs(a.X-b.X) + abs(a.Y-b.Y)
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Point represents a 2D point with integer coordinates
type Point struct {
	X, Y int
}

// ResetTurn resets the turn processed flag to allow AI processing in the next turn
func (s *AISystem) ResetTurn() {
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
