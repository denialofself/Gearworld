package screens

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/systems"
)

// DebugScreen shows debug messages in a modal window
type DebugScreen struct {
	*BaseScreen
	scrollOffset int
	width        int
	height       int
	background   color.Color
	textColor    color.Color
}

// NewDebugScreen creates a new debug screen
func NewDebugScreen() *DebugScreen {
	return &DebugScreen{
		BaseScreen:   NewBaseScreen(),
		scrollOffset: 0,
		width:        600,
		height:       400,
		background:   color.RGBA{0, 0, 0, 255}, // Solid black
		textColor:    color.White,
	}
}

// Update handles input for the debug screen
func (s *DebugScreen) Update() error {
	// Handle scrolling through debug messages with arrow keys
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.scrollUp()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.scrollDown()
	}

	// ESC to close debug window
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ErrCloseScreen
	}

	return nil
}

// scrollUp moves the view up by one line
func (s *DebugScreen) scrollUp() {
	if s.scrollOffset > 0 {
		s.scrollOffset--
	}
}

// scrollDown moves the view down by one line
func (s *DebugScreen) scrollDown() {
	// Get the debug log
	debugLog := systems.GetDebugLog()
	if s.scrollOffset < len(debugLog.Messages)-1 {
		s.scrollOffset++
	}
}

// Draw renders the debug screen
func (s *DebugScreen) Draw(screen *ebiten.Image) {
	// Calculate center position
	screenWidth, screenHeight := screen.Size()
	x := (screenWidth - s.width) / 2
	y := (screenHeight - s.height) / 2

	// Draw semi-transparent background
	modal := ebiten.NewImage(s.width, s.height)
	modal.Fill(s.background)

	// Draw frame
	frameWidth := 2.0
	ebitenutil.DrawRect(modal, 0, 0, frameWidth, float64(s.height), color.White)                           // Left
	ebitenutil.DrawRect(modal, float64(s.width)-frameWidth, 0, frameWidth, float64(s.height), color.White) // Right
	ebitenutil.DrawRect(modal, 0, 0, float64(s.width), frameWidth, color.White)                            // Top
	ebitenutil.DrawRect(modal, 0, float64(s.height)-frameWidth, float64(s.width), frameWidth, color.White) // Bottom

	// Convert text color to RGBA
	textRGBA := color.RGBAModel.Convert(s.textColor).(color.RGBA)

	// Draw title
	title := "DEBUG LOG"
	titleX := (s.width - len(title)*6) / 2 // Approximate text width
	titleImg := ebiten.NewImage(s.width, 20)
	ebitenutil.DebugPrintAt(titleImg, title, titleX, 0)
	// Apply title color
	titleOp := &ebiten.DrawImageOptions{}
	titleOp.ColorM.Scale(
		float64(textRGBA.R)/255.0,
		float64(textRGBA.G)/255.0,
		float64(textRGBA.B)/255.0,
		1.0,
	)
	modal.DrawImage(titleImg, titleOp)

	// Draw debug messages
	debugLog := systems.GetDebugLog()
	messages := debugLog.Messages
	startY := 30
	lineHeight := 16
	maxLines := (s.height - startY) / lineHeight

	// Calculate visible range
	startIdx := s.scrollOffset
	if startIdx > len(messages)-maxLines {
		startIdx = len(messages) - maxLines
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Draw visible messages
	for i := 0; i < maxLines && startIdx+i < len(messages); i++ {
		msg := messages[startIdx+i]
		// Use the message's color
		msgColor := color.RGBAModel.Convert(msg.GetColor()).(color.RGBA)
		// Create a new image for this line to draw with the correct color
		lineImg := ebiten.NewImage(s.width, lineHeight)
		ebitenutil.DebugPrintAt(lineImg, msg.Text, 10, 0)
		// Draw the line with the correct color
		op := &ebiten.DrawImageOptions{}
		op.ColorM.Scale(
			float64(msgColor.R)/255.0,
			float64(msgColor.G)/255.0,
			float64(msgColor.B)/255.0,
			1.0,
		)
		// Position the line
		op.GeoM.Translate(0, float64(startY+i*lineHeight))
		modal.DrawImage(lineImg, op)
	}

	// Draw scroll indicator if needed
	if len(messages) > maxLines {
		scrollBarHeight := float64(maxLines) / float64(len(messages)) * float64(s.height-startY)
		scrollBarY := float64(startY) + float64(s.scrollOffset)/float64(len(messages))*float64(s.height-startY)
		ebitenutil.DrawRect(modal, float64(s.width-10), scrollBarY, 5, scrollBarHeight, color.White)
	}

	// Draw controls
	controlsY := s.height - 20
	controlsImg := ebiten.NewImage(s.width, 20)
	ebitenutil.DebugPrintAt(controlsImg, "↑/↓: Scroll  ESC: Close", 10, 0)
	// Apply controls color
	controlsOp := &ebiten.DrawImageOptions{}
	controlsOp.ColorM.Scale(
		float64(textRGBA.R)/255.0,
		float64(textRGBA.G)/255.0,
		float64(textRGBA.B)/255.0,
		1.0,
	)
	controlsOp.GeoM.Translate(0, float64(controlsY))
	modal.DrawImage(controlsImg, controlsOp)

	// Draw the modal to the screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(modal, op)
}

// Layout implements the Screen interface
func (s *DebugScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// ErrCloseScreen is returned when the screen should be closed
var ErrCloseScreen = error(nil)
