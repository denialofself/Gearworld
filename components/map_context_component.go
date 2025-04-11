package components

import "ebiten-rogue/ecs"

// MapContextComponent identifies which map an entity belongs to
type MapContextComponent struct {
	MapID ecs.EntityID // The ID of the map entity this entity belongs to
}

// NewMapContextComponent creates a new map context component
func NewMapContextComponent(mapID ecs.EntityID) *MapContextComponent {
	return &MapContextComponent{
		MapID: mapID,
	}
}

// Define the component ID constant explicitly here as well
const MapContextID = ecs.ComponentID(12)
