package server

import (
	"sync"
)

// Event represents an event that can be published to subscribers
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// EventBus manages event publishing and subscriptions for SSE
type EventBus struct {
	subscribers []chan Event
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make([]chan Event, 0),
	}
}

// Subscribe creates a new subscription channel
func (eb *EventBus) Subscribe() chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	// Create buffered channel to prevent blocking
	ch := make(chan Event, 100)
	eb.subscribers = append(eb.subscribers, ch)
	
	return ch
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(ch chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	// Find and remove the channel
	for i, subscriber := range eb.subscribers {
		if subscriber == ch {
			// Remove from slice
			eb.subscribers = append(eb.subscribers[:i], eb.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// Publish sends an event to all subscribers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	// Send to all subscribers
	for _, ch := range eb.subscribers {
		// Non-blocking send
		select {
		case ch <- event:
		default:
			// Channel is full, skip this event
			// In production, might want to handle this differently
		}
	}
}