package config

// Screen layout configuration
const (
	// Tile size in pixels - source tiles are 12x12, this scales them up
	TileSize = 12

	// Window dimensions in tiles
	ScreenWidth  = 85
	ScreenHeight = 65

	// UI layout
	GameScreenWidth  = 50 // Game area width in tiles (reduced from 54 to give more space to stats)
	GameScreenHeight = 40 // Game area height in tiles

	// Message window configuration
	MessageWindowHeight = 8                                  // Reduced from full height to 8 tiles
	MessageWindowStartY = ScreenHeight - MessageWindowHeight // Start position of message window

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
	return WindowWidth, WindowHeight // Return the actual calculated dimensions
}
