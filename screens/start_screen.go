package screens

import (
	"errors"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/config"
	"ebiten-rogue/systems"
)

// Error constants for screen transitions
var (
	ErrNewGame  = errors.New("new game")
	ErrLoadGame = errors.New("load game")
	ErrOptions  = errors.New("options")
	ErrQuit     = errors.New("quit")
)

// StartScreen handles the game's start menu
type StartScreen struct {
	*BaseScreen
	selectedOption int
	options        []string
	titleColor     color.Color
	optionColor    color.Color
	selectedColor  color.Color
	backgroundImg  *ebiten.Image
	audioSystem    *systems.AudioSystem
}

// NewStartScreen creates a new start screen
func NewStartScreen(audioSystem *systems.AudioSystem) *StartScreen {
	// Load background image
	img, _, err := ebitenutil.NewImageFromFile("assets/start_screen.png")
	if err != nil {
		log.Fatalf("Failed to load start screen image: %v", err)
	}

	return &StartScreen{
		BaseScreen:     NewBaseScreen(),
		selectedOption: 0,
		options: []string{
			"New Game",
			"Load Game",
			"Options",
			"Quit",
		},
		titleColor:    color.RGBA{255, 230, 150, 255}, // Gold
		optionColor:   color.RGBA{200, 200, 200, 255}, // Light Gray
		selectedColor: color.RGBA{255, 255, 255, 255}, // White
		backgroundImg: img,
		audioSystem:   audioSystem,
	}
}

// Update handles input for the start screen
func (s *StartScreen) Update() error {
	// Start playing background music if not already playing
	if !s.audioSystem.IsBGMPlaying() {
		s.audioSystem.PlayBGM("assets/audio/background.mp3")
	}

	// Handle arrow key navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		s.selectedOption = (s.selectedOption - 1 + len(s.options)) % len(s.options)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		s.selectedOption = (s.selectedOption + 1) % len(s.options)
	}

	// Handle selection
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		switch s.selectedOption {
		case 0: // New Game
			return ErrNewGame
		case 1: // Load Game
			return ErrLoadGame
		case 2: // Options
			return ErrOptions
		case 3: // Quit
			return ErrQuit
		}
	}

	return nil
}

// Draw renders the start screen
func (s *StartScreen) Draw(screen *ebiten.Image) {
	// Draw background image
	screenWidth, screenHeight := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(
		float64(screenWidth)/float64(s.backgroundImg.Bounds().Dx()),
		float64(screenHeight)/float64(s.backgroundImg.Bounds().Dy()),
	)
	screen.DrawImage(s.backgroundImg, op)

	// Calculate center position
	centerX := screenWidth / 2
	centerY := screenHeight / 2

	// Draw options
	optionSpacing := 30
	startY := centerY - (len(s.options)*optionSpacing)/2

	for i, option := range s.options {
		y := startY + i*optionSpacing
		optionX := centerX - (len(option)*6)/2

		// Choose color based on selection
		textColor := s.optionColor
		if i == s.selectedOption {
			textColor = s.selectedColor
		}

		// Create a new image for this line to draw with the correct color
		lineImg := ebiten.NewImage(screenWidth, optionSpacing)
		ebitenutil.DebugPrintAt(lineImg, option, optionX, 0)

		// Create a colored version of the line
		coloredLine := ebiten.NewImage(screenWidth, optionSpacing)
		op := &ebiten.DrawImageOptions{}
		op.ColorM.Scale(
			float64(textColor.(color.RGBA).R)/255.0,
			float64(textColor.(color.RGBA).G)/255.0,
			float64(textColor.(color.RGBA).B)/255.0,
			1.0,
		)
		coloredLine.DrawImage(lineImg, op)

		// Position and draw the colored line
		lineOp := &ebiten.DrawImageOptions{}
		lineOp.GeoM.Translate(0, float64(y))
		screen.DrawImage(coloredLine, lineOp)
	}
}

// Layout implements the Screen interface
func (s *StartScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}
