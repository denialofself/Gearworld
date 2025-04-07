package config

// Screen layout configuration
const (
	// Tile size in pixels
	TileSize = 16

	// Window dimensions in tiles
	ScreenWidth  = 64
	ScreenHeight = 48

	// UI layout
	GameScreenWidth  = 50 // Game area width in tiles (reduced from 54 to give more space to stats)
	GameScreenHeight = 40 // Game area height in tiles

	// Window dimensions in pixels (derived from tile dimensions)
	WindowWidth  = ScreenWidth * TileSize
	WindowHeight = ScreenHeight * TileSize
)

// GetScreenDimensions returns the screen dimensions in pixels
func GetScreenDimensions() (width, height int) {
	return WindowWidth, WindowHeight
}

// GetWindowSize returns the recommended window size (may be different from actual screen dimensions)
func GetWindowSize() (width, height int) {
	return 1024, 768 // Can be adjusted if needed for UI scaling
}
