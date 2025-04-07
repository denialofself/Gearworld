package systems

// MessageLog stores game messages
type MessageLog struct {
	Messages    []string
	MaxMessages int
}

// Global message log instance (singleton)
var globalMessageLog *MessageLog

// GetMessageLog returns the global message log instance
func GetMessageLog() *MessageLog {
	if globalMessageLog == nil {
		globalMessageLog = NewMessageLog()
	}
	return globalMessageLog
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
