package systems

import (
	"image"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tileset handles loading and drawing the tile spritesheet
type Tileset struct {
	Image    *ebiten.Image
	TileSize int
	Width    int // Number of tiles horizontally in the tileset
	Height   int // Number of tiles vertically in the tileset
}

// NewTileset loads a tileset from a file
func NewTileset(filename string, tileSize int) (*Tileset, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// Convert to ebiten image
	ebitenImage := ebiten.NewImageFromImage(img)

	// Calculate the dimensions in tiles
	bounds := ebitenImage.Bounds()
	srcTileSize := 12 // The source tileset has 12x12 pixel tiles
	widthInTiles := bounds.Dx() / srcTileSize
	heightInTiles := bounds.Dy() / srcTileSize

	return &Tileset{
		Image:    ebitenImage,
		TileSize: tileSize,
		Width:    widthInTiles,
		Height:   heightInTiles,
	}, nil
}

// GetTileCoords returns the x, y coordinates of a tile in the tileset
// based on the ASCII value of the character
func (t *Tileset) GetTileCoords(char rune) (int, int) {
	index := int(char)
	// Code Page 437 tileset layout
	x := index % 16
	y := index / 16
	return x, y
}

// TileID represents a tile by its position in the tileset
type TileID struct {
	X, Y int
}

// NewTileID creates a TileID from x,y coordinates in the tileset
func NewTileID(x, y int) TileID {
	return TileID{X: x, Y: y}
}

// DrawTileByID draws a tile specified by its position in the tileset
func (t *Tileset) DrawTileByID(target *ebiten.Image, tileID TileID, x, y int, clr color.Color) {
	// Ensure the tile ID is within bounds
	if tileID.X < 0 || tileID.X >= t.Width || tileID.Y < 0 || tileID.Y >= t.Height {
		// Draw a default "unknown" tile or return
		t.DrawTile(target, '?', x, y, color.RGBA{255, 0, 255, 255})
		return
	}

	// Calculate source rectangle in the tileset
	srcTileSize := 12 // The actual size in the PNG is 12x12
	sx := tileID.X * srcTileSize
	sy := tileID.Y * srcTileSize

	// Calculate destination position on the screen
	dx := float64(x * t.TileSize)
	dy := float64(y * t.TileSize)

	// Set up draw options
	op := &ebiten.DrawImageOptions{}

	// Scale the tile to fit our tile size (if different from source size)
	scaleX := float64(t.TileSize) / float64(srcTileSize)
	scaleY := float64(t.TileSize) / float64(srcTileSize)
	op.GeoM.Scale(scaleX, scaleY)

	// Apply color
	if clr != nil {
		r, g, b, a := clr.RGBA()
		rf := float64(r) / 0xffff
		gf := float64(g) / 0xffff
		bf := float64(b) / 0xffff
		af := float64(a) / 0xffff

		op.ColorM.Scale(rf, gf, bf, af)
	}

	// Set destination position (after scaling)
	op.GeoM.Translate(dx, dy)

	// Draw the tile
	rect := image.Rect(sx, sy, sx+srcTileSize, sy+srcTileSize)
	target.DrawImage(t.Image.SubImage(rect).(*ebiten.Image), op)
}

// DrawTile draws a single tile on the screen
func (t *Tileset) DrawTile(target *ebiten.Image, char rune, x, y int, clr color.Color) {
	tileX, tileY := t.GetTileCoords(char)
	t.DrawTileByID(target, TileID{X: tileX, Y: tileY}, x, y, clr)
}

// DrawString draws a string of characters
func (t *Tileset) DrawString(target *ebiten.Image, text string, x, y int, clr color.Color) {
	for i, char := range text {
		t.DrawTile(target, char, x+i, y, clr)
	}
}
