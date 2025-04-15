package screens

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Screen represents a game screen that can be pushed onto the screen stack
type Screen interface {
	// Update updates the screen state
	Update() error
	// Draw draws the screen
	Draw(screen *ebiten.Image)
	// Layout handles screen layout
	Layout(outsideWidth, outsideHeight int) (int, int)
}

// ScreenStack manages a stack of screens
type ScreenStack struct {
	screens []Screen
}

// NewScreenStack creates a new screen stack
func NewScreenStack() *ScreenStack {
	return &ScreenStack{
		screens: make([]Screen, 0),
	}
}

// Push adds a new screen to the top of the stack
func (s *ScreenStack) Push(screen Screen) {
	s.screens = append(s.screens, screen)
}

// Pop removes the top screen from the stack
func (s *ScreenStack) Pop() Screen {
	if len(s.screens) == 0 {
		return nil
	}
	top := s.screens[len(s.screens)-1]
	s.screens = s.screens[:len(s.screens)-1]
	return top
}

// Peek returns the top screen without removing it
func (s *ScreenStack) Peek() Screen {
	if len(s.screens) == 0 {
		return nil
	}
	return s.screens[len(s.screens)-1]
}

// Update updates the top screen
func (s *ScreenStack) Update() error {
	if top := s.Peek(); top != nil {
		return top.Update()
	}
	return nil
}

// Draw draws all screens from bottom to top
func (s *ScreenStack) Draw(screen *ebiten.Image) {
	for _, scr := range s.screens {
		scr.Draw(screen)
	}
}

// Layout handles layout for the top screen
func (s *ScreenStack) Layout(outsideWidth, outsideHeight int) (int, int) {
	if top := s.Peek(); top != nil {
		return top.Layout(outsideWidth, outsideHeight)
	}
	return outsideWidth, outsideHeight
}
