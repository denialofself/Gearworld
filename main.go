package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"ebiten-rogue/config"
)

func main() {
	// First check if we should run the tileset viewer
	if len(os.Args) > 1 && os.Args[1] == "--view-tileset" {
		// Run the tileset viewer
		viewer := NewTilesetViewer("Nice_curses_12x12.png", 36) // Use a larger tile size for better visibility
		ebiten.SetWindowSize(800, 600)
		ebiten.SetWindowTitle("Tileset Viewer - Nice_curses_12x12.png")
		if err := ebiten.RunGame(viewer); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Run the normal game
	game := NewGame()

	// Get window size from config
	windowWidth, windowHeight := config.GetWindowSize()
	ebiten.SetWindowSize(windowWidth, windowHeight)

	ebiten.SetWindowTitle("Ebiten Roguelike")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
