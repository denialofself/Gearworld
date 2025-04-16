package screens

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"ebiten-rogue/config"
)

// GameOverScreen displays the game over message
type GameOverScreen struct {
	*BaseScreen
}

// NewGameOverScreen creates a new game over screen
func NewGameOverScreen() *GameOverScreen {
	return &GameOverScreen{
		BaseScreen: NewBaseScreen(),
	}
}

// Update handles input for the game over screen
func (s *GameOverScreen) Update() error {
	return nil
}

// Draw draws the game over screen
func (s *GameOverScreen) Draw(screen *ebiten.Image) {
	// Draw game over message
	screenWidth, screenHeight := screen.Size()
	text := "Game Over!\n\nPress Escape to return to the start screen"
	ebitenutil.DebugPrintAt(screen, text, screenWidth/2-100, screenHeight/2-20)
}

// Layout implements the Screen interface
func (s *GameOverScreen) Layout(outsideWidth, outsideHeight int) (int, int) {
	return config.ScreenWidth * config.TileSize, config.ScreenHeight * config.TileSize
}
