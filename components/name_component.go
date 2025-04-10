package components

// NameComponent stores the display name for entities
type NameComponent struct {
	Name string
}

// NewNameComponent creates a new name component
func NewNameComponent(name string) *NameComponent {
	return &NameComponent{
		Name: name,
	}
}
