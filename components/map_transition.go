package components

// MapTransitionComponent stores information about transitioning between maps
type MapTransitionComponent struct {
	TransitionType     int    // Type of transition (e.g., stairs up, stairs down)
	DestinationMapType string // Type of map to transition to (e.g., "worldmap", "dungeon")
	DestinationX       int    // X position in destination map
	DestinationY       int    // Y position in destination map
}

// Transition types
const (
	TransitionStairsDown = iota
	TransitionStairsUp
	TransitionPortal
)

// MapTypeComponent identifies what kind of map an entity represents
type MapTypeComponent struct {
	MapType string // "worldmap" or "dungeon"
	Level   int    // For dungeons, indicates the depth
}

// Position ID for the MapTransitionComponent
const MapTransition = "map_transition"


// NewMapTransitionComponent creates a new map transition component
func NewMapTransitionComponent(transitionType int, destMapType string, destX, destY int) *MapTransitionComponent {
	return &MapTransitionComponent{
		TransitionType:     transitionType,
		DestinationMapType: destMapType,
		DestinationX:       destX,
		DestinationY:       destY,
	}
}

// NewMapTypeComponent creates a new map type component
func NewMapTypeComponent(mapType string, level int) *MapTypeComponent {
	return &MapTypeComponent{
		MapType: mapType,
		Level:   level,
	}
}
