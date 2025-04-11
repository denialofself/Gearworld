package systems

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// MessageLog stores game messages
type MessageLog struct {
	Messages    []string
	MaxMessages int
}

// Global message log instance (singleton)
var globalMessageLog *MessageLog
var globalDebugLog *MessageLog

// Debug log file writer
var debugLogWriter io.Writer

// SetDebugLogWriter sets a writer for debug log messages (typically a file)
func SetDebugLogWriter(writer io.Writer) {
	debugLogWriter = writer
}

// GetMessageLog returns the global message log instance
func GetMessageLog() *MessageLog {
	if globalMessageLog == nil {
		globalMessageLog = NewMessageLog()
	}
	return globalMessageLog
}

// GetDebugLog returns the global debug log instance
func GetDebugLog() *MessageLog {
	if globalDebugLog == nil {
		globalDebugLog = NewMessageLog()
	}
	return globalDebugLog
}

// NewMessageLog creates a new message log
func NewMessageLog() *MessageLog {
	return &MessageLog{
		Messages:    []string{},
		MaxMessages: 100, // Store the last 100 messages
	}
}

// Add adds a message to the log
func (ml *MessageLog) Add(message string) {
	// If this is the main message log, check if it's a debug message
	// and if so, route it to the debug log instead
	if ml == globalMessageLog && strings.HasPrefix(message, "DEBUG:") {
		GetDebugLog().Add(message)
		return
	}

	// Add timestamp for debug log messages if it's the debug log
	if ml == globalDebugLog && debugLogWriter != nil {
		// Format with timestamp for file logging
		timestamp := time.Now().Format("15:04:05.000")
		formattedMsg := fmt.Sprintf("[%s] %s\n", timestamp, message)

		// Write to the debug log file
		_, err := fmt.Fprint(debugLogWriter, formattedMsg)
		if err != nil {
			// If we can't write to file, print the error to console
			fmt.Fprintf(os.Stderr, "Error writing to debug log: %v\n", err)
		}
	}

	ml.Messages = append(ml.Messages, message)

	// Truncate if we have too many messages
	if len(ml.Messages) > ml.MaxMessages {
		ml.Messages = ml.Messages[len(ml.Messages)-ml.MaxMessages:]
	}
}

// RecentMessages gets the n most recent messages
func (ml *MessageLog) RecentMessages(n int) []string {
	if n > len(ml.Messages) {
		n = len(ml.Messages)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		// Get messages from newest to oldest
		result[i] = ml.Messages[len(ml.Messages)-1-i]
	}

	return result
}

// Clear clears all messages
func (ml *MessageLog) Clear() {
	ml.Messages = []string{}
}
