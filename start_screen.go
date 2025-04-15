package main

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"ebiten-rogue/systems"
)

// GameState represents the current state of the game
type GameState int

const (
	StateStartScreen GameState = iota
	StatePlaying
)

// StartScreen handles the title screen state
type StartScreen struct {
	titleImage  *ebiten.Image
	audioSystem *systems.AudioSystem
}

// NewStartScreen creates a new start screen
func NewStartScreen(audioSystem *systems.AudioSystem) *StartScreen {
	var titleImage *ebiten.Image
	var err error

	// Try different image formats
	formats := []string{"png", "jpg", "jpeg", "gif"}
	for _, format := range formats {
		path := fmt.Sprintf("assets/start_screen.%s", format)
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			continue
		}

		titleImage, _, err = ebitenutil.NewImageFromFile(path)
		if err == nil {
			systems.GetDebugLog().Add(fmt.Sprintf("Successfully loaded title screen image: %s", path))
			break
		}
	}

	// If all formats failed, create a fallback
	if err != nil {
		systems.GetDebugLog().Add("Failed to load title screen image. Tried formats: png, jpg, jpeg, gif")
		titleImage = ebiten.NewImage(800, 600)
		titleImage.Fill(color.RGBA{20, 20, 40, 255})
	}

	// Scale down the image to 75% of its original size
	scaledWidth := int(float64(titleImage.Bounds().Dx()) * 0.50)
	scaledHeight := int(float64(titleImage.Bounds().Dy()) * 0.50)
	scaledImage := ebiten.NewImage(scaledWidth, scaledHeight)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.50, 0.50)
	scaledImage.DrawImage(titleImage, op)

	return &StartScreen{
		titleImage:  scaledImage,
		audioSystem: audioSystem,
	}
}

// Update handles input for the start screen
func (s *StartScreen) Update() bool {
	// Check if background music is playing, if not start it
	if s.audioSystem != nil && !s.audioSystem.IsBGMPlaying() {
		// Play the background music directly from file
		err := s.audioSystem.PlayBGM("assets/audio/background.mp3")
		if err != nil {
			systems.GetDebugLog().Add("Failed to play background music: " + err.Error())
		}
	}

	// Check for Enter key press to start the game
	return inpututil.IsKeyJustPressed(ebiten.KeyEnter)
}

// Draw renders the start screen
func (s *StartScreen) Draw(screen *ebiten.Image) {
	// Get screen dimensions
	screenWidth, screenHeight := screen.Size()

	// Calculate center position for the title image
	titleWidth, titleHeight := s.titleImage.Size()
	titleX := (screenWidth - titleWidth) / 2
	titleY := (screenHeight - titleHeight) / 2

	// Draw the scaled image centered
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(titleX), float64(titleY))
	screen.DrawImage(s.titleImage, op)

	// Draw "Press Enter to Start" text below the image
	text := "Press Enter to Start"
	textWidth := len(text) * 12
	textX := (screenWidth - textWidth) / 2
	textY := titleY + titleHeight + 20 // Position 20 pixels below the image

	// Draw solid background for text
	textBg := ebiten.NewImage(textWidth+40, 30)
	textBg.Fill(color.RGBA{0, 0, 0, 255})
	textBgOp := &ebiten.DrawImageOptions{}
	textBgOp.GeoM.Translate(float64(textX-20), float64(textY-5))
	screen.DrawImage(textBg, textBgOp)

	// Draw the text
	for i, c := range text {
		charX := textX + (i * 12)
		ebitenutil.DebugPrintAt(screen, string(c), charX, textY)
	}
}

// Layout implements ebiten.Game's Layout
func (s *StartScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
