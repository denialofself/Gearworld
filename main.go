package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"ebiten-rogue/config"
	"ebiten-rogue/systems"
)

// setupFileLogging configures debug logging to write to a file
func setupFileLogging(filepath string) error {
	// Create the log file (append if exists, create if it doesn't)
	logFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// Redirect our debug logging system to use this file
	systems.SetDebugLogWriter(logFile)

	// Write initial timestamp and header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logFile.WriteString("===== DEBUG LOG STARTED AT " + timestamp + " =====\n")

	// Add logging info to console as well
	log.Printf("Debug logging enabled. Writing logs to: %s", filepath)

	return nil
}

func main() {
	// Define command-line flags
	debugLogFile := flag.String("log", "", "Filename to write debug logs to")
	viewTileset := flag.Bool("view-tileset", false, "Run the tileset viewer")
	worldMap := flag.Bool("world-map", false, "Run the world map tester")

	// Parse the command line flags
	flag.Parse()

	// Set up debug file logging if enabled
	if *debugLogFile != "" {
		if err := setupFileLogging(*debugLogFile); err != nil {
			log.Printf("Error setting up debug logging: %v", err)
		}
	}

	// Handle the special modes
	if *viewTileset {
		// Run the tileset viewer
		viewer := NewTilesetViewer("Nice_curses_12x12.png", 36) // Use a larger tile size for better visibility
		ebiten.SetWindowSize(800, 600)
		ebiten.SetWindowTitle("Tileset Viewer - Nice_curses_12x12.png")
		if err := ebiten.RunGame(viewer); err != nil {
			log.Fatal(err)
		}
		return
	} else if *worldMap {
		// Run the specialized world map tester
		worldMapTester := NewWorldMapTester()

		// Get window size from config
		windowWidth, windowHeight := config.GetWindowSize()
		ebiten.SetWindowSize(windowWidth, windowHeight)
		ebiten.SetWindowTitle("Ebiten Roguelike - World Map Tester")
		if err := ebiten.RunGame(worldMapTester); err != nil {
			log.Fatal(err)
		}
		return
	}

	// For backward compatibility with the old command line format
	if len(os.Args) > 1 && flag.NArg() > 0 {
		firstArg := flag.Arg(0)
		if firstArg == "--view-tileset" {
			// Run the tileset viewer
			viewer := NewTilesetViewer("Nice_curses_12x12.png", 36)
			ebiten.SetWindowSize(800, 600)
			ebiten.SetWindowTitle("Tileset Viewer - Nice_curses_12x12.png")
			if err := ebiten.RunGame(viewer); err != nil {
				log.Fatal(err)
			}
			return
		} else if firstArg == "--world-map" {
			// Run the specialized world map tester
			worldMapTester := NewWorldMapTester()
			windowWidth, windowHeight := config.GetWindowSize()
			ebiten.SetWindowSize(windowWidth, windowHeight)
			ebiten.SetWindowTitle("Ebiten Roguelike - World Map Tester")
			if err := ebiten.RunGame(worldMapTester); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	// Run the normal game (dungeon mode)
	game := NewGame()
	// Get window size from config
	windowWidth, windowHeight := config.GetWindowSize()
	ebiten.SetWindowSize(windowWidth, windowHeight)

	// Enable fullscreen mode
	ebiten.SetFullscreen(true)

	ebiten.SetWindowTitle("Ebiten Roguelike")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
