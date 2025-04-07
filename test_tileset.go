package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"ebiten-rogue/systems"
)

// TilesetViewer implements ebiten.Game interface.
type TilesetViewer struct {
	tileset       *systems.Tileset
	screenWidth   int
	screenHeight  int
	tileSize      int
	displayWidth  int    // How many tiles to display horizontally
	displayHeight int    // How many tiles to display vertically
	offsetX       int    // Scrolling offset for viewing all tiles
	offsetY       int    // Scrolling offset for viewing all tiles
	filename      string // Tileset filename
}

// NewTilesetViewer creates a new tileset viewer
func NewTilesetViewer(filename string, tileSize int) *TilesetViewer {
	// Create the tileset
	tileset, err := systems.NewTileset(filename, tileSize)
	if err != nil {
		panic(err)
	}

	// Calculate how many tiles we can fit on screen
	displayWidth := 16  // Display 16 tiles across
	displayHeight := 12 // Display 12 tiles down

	return &TilesetViewer{
		tileset:       tileset,
		tileSize:      tileSize,
		screenWidth:   displayWidth*tileSize + 50,   // Add some margin
		screenHeight:  displayHeight*tileSize + 120, // Add space for header and footer
		displayWidth:  displayWidth,
		displayHeight: displayHeight,
		offsetX:       0,
		offsetY:       0,
		filename:      filename,
	}
}

// Update handles input for scrolling
func (t *TilesetViewer) Update() error {
	// Handle keyboard for scrolling
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		if t.offsetX < t.tileset.Width-t.displayWidth {
			t.offsetX++
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && t.offsetX > 0 {
		t.offsetX--
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		if t.offsetY < t.tileset.Height-t.displayHeight {
			t.offsetY++
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && t.offsetY > 0 {
		t.offsetY--
	}

	// Page navigation
	if ebiten.IsKeyPressed(ebiten.KeyPageDown) {
		t.offsetY += t.displayHeight
		if t.offsetY > t.tileset.Height-t.displayHeight {
			t.offsetY = t.tileset.Height - t.displayHeight
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyPageUp) {
		t.offsetY -= t.displayHeight
		if t.offsetY < 0 {
			t.offsetY = 0
		}
	}

	return nil
}

// Draw displays all the tiles with their coordinates
func (t *TilesetViewer) Draw(screen *ebiten.Image) {
	// Clear the screen
	screen.Fill(color.RGBA{30, 30, 30, 255})

	// Display information about the tileset
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tileset: %s", t.filename), 10, 10)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Size: %dx%d tiles", t.tileset.Width, t.tileset.Height), 10, 30)
	ebitenutil.DebugPrintAt(screen, "Use arrow keys to scroll, Page Up/Down to navigate faster", 10, 50)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Viewing offset: %d,%d", t.offsetX, t.offsetY), 10, 70)

	// Draw a grid of tiles with their coordinates
	for y := 0; y < t.displayHeight; y++ {
		for x := 0; x < t.displayWidth; x++ {
			// Calculate the actual tile coordinates including the offset
			tileX := x + t.offsetX
			tileY := y + t.offsetY

			// Skip if outside the tileset bounds
			if tileX >= t.tileset.Width || tileY >= t.tileset.Height {
				continue
			}

			// Calculate screen position (where to draw on screen)
			screenX := x * t.tileSize
			screenY := y*t.tileSize + 100 // Add vertical offset for the header text

			// Draw a background box
			ebitenutil.DrawRect(screen, float64(screenX), float64(screenY),
				float64(t.tileSize), float64(t.tileSize), color.RGBA{60, 60, 60, 255})

			// Draw the tile directly from the tileset using its position

			// Set up tile scaling and positioning
			srcTileSize := 12 // The source tileset has 12x12 pixel tiles
			sx := tileX * srcTileSize
			sy := tileY * srcTileSize

			// Draw the tile
			op := &ebiten.DrawImageOptions{}

			// Scale to our display tile size
			scaleX := float64(t.tileSize) / float64(srcTileSize)
			scaleY := float64(t.tileSize) / float64(srcTileSize)
			op.GeoM.Scale(scaleX, scaleY)

			// Move to the correct position on screen
			op.GeoM.Translate(float64(screenX), float64(screenY))

			// Get the tile from the source image and draw it
			rect := image.Rect(sx, sy, sx+srcTileSize, sy+srcTileSize)
			screen.DrawImage(t.tileset.Image.SubImage(rect).(*ebiten.Image), op)

			// Calculate the ASCII value if this position were mapped to a character
			asciiValue := 0
			if tileY < 16 && tileX < 16 { // Standard CP437 layout
				asciiValue = tileY*16 + tileX
			}

			// Draw the coordinates below the tile
			posText := fmt.Sprintf("(%d,%d)", tileX, tileY)
			ebitenutil.DebugPrintAt(screen, posText, screenX+2, screenY+t.tileSize-24)

			// Add ASCII info if relevant
			if asciiValue > 0 && asciiValue < 256 {
				asciiInfo := fmt.Sprintf("#%d", asciiValue)
				if asciiValue >= 32 && asciiValue <= 126 {
					// Add representation for printable ASCII
					asciiInfo += " " + string(rune(asciiValue))
				}
				ebitenutil.DebugPrintAt(screen, asciiInfo, screenX+2, screenY+t.tileSize-12)
			}
		}
	}

	// Draw instructions at the bottom
	ebitenutil.DebugPrintAt(screen, "ESC: Return to game | Arrow keys: Navigate | Page Up/Down: Fast navigation", 10, t.screenHeight-20)
}

// Layout implements ebiten.Game's Layout.
func (t *TilesetViewer) Layout(outsideWidth, outsideHeight int) (int, int) {
	return t.screenWidth, t.screenHeight
}
