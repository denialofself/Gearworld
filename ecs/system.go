package ecs

// System defines an interface for processing entities with specific components
type System interface {
	// Update is called each frame to process entities
	Update(world *World, dt float64)
}
