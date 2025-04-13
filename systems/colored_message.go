package systems

import (
	"image/color"
)

// MessageType defines different types of messages that can appear in the log
type MessageType int

const (
	// MessageTypeNormal is for standard game messages (white/gray)
	MessageTypeNormal MessageType = iota
	// MessageTypeEnvironment is for descriptive or environmental text (gold)
	MessageTypeEnvironment
	// MessageTypeCombat is for combat messages (red)
	MessageTypeCombat
	// MessageTypeItem is for item-related messages (blue)
	MessageTypeItem
	// MessageTypeAlert is for important alerts (bright yellow)
	MessageTypeAlert
	// MessageTypeSystem is for system messages (purple/magenta)
	MessageTypeSystem
)

// ColoredMessage stores a message with its associated color
type ColoredMessage struct {
	Text string
	Type MessageType
}

// GetColor returns the color for the message based on its type
func (cm ColoredMessage) GetColor() color.RGBA {
	switch cm.Type {
	case MessageTypeEnvironment:
		return color.RGBA{218, 165, 32, 255} // Gold
	case MessageTypeCombat:
		return color.RGBA{255, 100, 100, 255} // Red
	case MessageTypeItem:
		return color.RGBA{100, 149, 237, 255} // Cornflower Blue
	case MessageTypeAlert:
		return color.RGBA{255, 255, 0, 255} // Bright Yellow
	case MessageTypeSystem:
		return color.RGBA{186, 85, 211, 255} // Medium Orchid (Purple)
	default:
		return color.RGBA{200, 200, 200, 255} // Light Gray (default)
	}
}
