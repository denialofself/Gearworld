package ecs

// ComponentID is a unique identifier for component types
type ComponentID uint

// Component is the base interface for all components
type Component interface{}

// ComponentMap stores components by their type ID
type ComponentMap map[ComponentID]Component
