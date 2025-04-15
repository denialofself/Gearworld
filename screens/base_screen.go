package screens

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// BaseScreen provides common functionality for all screens
type BaseScreen struct {
	// Screen dimensions
	width  int
	height int
}

// NewBaseScreen creates a new base screen
func NewBaseScreen() *BaseScreen {
	return &BaseScreen{}
}

// Update implements the Screen interface
func (s *BaseScreen) Update() error {
	return nil
}

// Draw implements the Screen interface
func (s *BaseScreen) Draw(screen *ebiten.Image) {
	// Base screen does nothing by default
}

// Layout implements the Screen interface
func (s *BaseScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	s.width = outsideWidth
	s.height = outsideHeight
	return outsideWidth, outsideHeight
}

// GetWidth returns the screen width
func (s *BaseScreen) GetWidth() int {
	return s.width
}

// GetHeight returns the screen height
func (s *BaseScreen) GetHeight() int {
	return s.height
}
