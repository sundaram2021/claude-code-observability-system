package service

import (
	"sync"

	"github.com/claude-code/observability/internal/domain"
)

// EventType represents the category of a published realtime event.
type EventType string

const (
	EventTraceCreated EventType = "TRACE_CREATED"
	EventSpanCreated  EventType = "SPAN_CREATED"
	EventSessionAlert EventType = "SESSION_ALERT"
)

// RealtimeEvent carries event data to subscribed SSE clients.
type RealtimeEvent struct {
	Type    EventType     `json:"type"`
	Trace   *domain.Trace `json:"trace,omitempty"`
	Span    *domain.Span  `json:"span,omitempty"`
	Message string        `json:"message,omitempty"`
}

// EventBus provides lightweight, non-blocking broadcast pub/sub.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan RealtimeEvent]struct{}
	bufferSize  int
}

// NewEventBus initializes a new pub/sub event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[chan RealtimeEvent]struct{}),
		bufferSize:  32,
	}
}

// Subscribe registers a new subscriber channel.
func (b *EventBus) Subscribe() chan RealtimeEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan RealtimeEvent, b.bufferSize)
	b.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (b *EventBus) Unsubscribe(ch chan RealtimeEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[ch]; exists {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Publish broadcasts an event non-blockingly to all active subscribers.
func (b *EventBus) Publish(event RealtimeEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// Non-blocking: drop if subscriber buffer is full to prevent pipeline stall
		}
	}
}

// SubscriberCount returns current active listener count.
func (b *EventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

