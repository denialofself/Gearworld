// filepath: d:\Temp\ebiten-rogue\ecs\event.go
package ecs

// EventType identifies different types of events
type EventType string

// Event interface that all events must implement
type Event interface {
	Type() EventType
}

// EventHandler is a function that processes events
type EventHandler func(Event)

// EventManager manages event subscriptions and dispatches
type EventManager struct {
	subscribers map[EventType][]EventHandler
}

// NewEventManager creates a new event manager
func NewEventManager() *EventManager {
	return &EventManager{
		subscribers: make(map[EventType][]EventHandler),
	}
}

// Subscribe registers a handler for a specific event type
func (em *EventManager) Subscribe(eventType EventType, handler EventHandler) {
	em.subscribers[eventType] = append(em.subscribers[eventType], handler)
}

// Unsubscribe removes a handler for a specific event type
func (em *EventManager) Unsubscribe(eventType EventType, handler EventHandler) {
	handlers, exists := em.subscribers[eventType]
	if !exists {
		return
	}

	// Create a new slice without the handler
	newHandlers := make([]EventHandler, 0, len(handlers))
	for _, h := range handlers {
		// This comparison is imperfect for functions, but it's the best we can do in Go
		// In practice, you might want to use a more robust identification mechanism
		if &h != &handler {
			newHandlers = append(newHandlers, h)
		}
	}

	if len(newHandlers) == 0 {
		delete(em.subscribers, eventType)
	} else {
		em.subscribers[eventType] = newHandlers
	}
}

// Emit dispatches an event to all subscribed handlers
func (em *EventManager) Emit(event Event) {
	eventType := event.Type()
	handlers, exists := em.subscribers[eventType]
	if !exists {
		return
	}

	for _, handler := range handlers {
		handler(event)
	}
}
