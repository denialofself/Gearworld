package screens

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ModalScreen represents a popup window that appears on top of other screens
type ModalScreen struct {
	*BaseScreen
	title      string
	content    string
	width      int
	height     int
	background color.Color
	textColor  color.Color
}

// NewModalScreen creates a new modal screen
func NewModalScreen(title, content string, width, height int) *ModalScreen {
	return &ModalScreen{
		BaseScreen: NewBaseScreen(),
		title:      title,
		content:    content,
		width:      width,
		height:     height,
		background: color.RGBA{0, 0, 0, 200}, // Semi-transparent black
		textColor:  color.White,
	}
}

// Draw implements the Screen interface
func (s *ModalScreen) Draw(screen *ebiten.Image) {
	// Calculate center position
	screenWidth, screenHeight := screen.Size()
	x := (screenWidth - s.width) / 2
	y := (screenHeight - s.height) / 2

	// Draw semi-transparent background
	modal := ebiten.NewImage(s.width, s.height)
	modal.Fill(s.background)

	// Draw border
	ebitenutil.DrawRect(modal, 0, 0, float64(s.width), float64(s.height), color.White)

	// Draw title
	titleX := (s.width - len(s.title)*6) / 2 // Approximate text width
	ebitenutil.DebugPrintAt(modal, s.title, titleX, 10)

	// Draw content
	ebitenutil.DebugPrintAt(modal, s.content, 10, 30)

	// Draw the modal to the screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(modal, op)
}

// Update implements the Screen interface
func (s *ModalScreen) Update() error {
	return nil
}

// Layout implements the Screen interface
func (s *ModalScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
