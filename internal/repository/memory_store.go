package repository

import (
	"sync"

	"github.com/claude-code/observability/internal/domain"
)

// MemoryStore provides low-footprint, in-memory bounded storage for traces and sessions.
type MemoryStore struct {
	mu           sync.RWMutex
	traces       *RingBuffer[*domain.Trace]
	traceIndex   map[string]*domain.Trace
	spanIndex    map[string]*domain.Span
	sessions     map[string]*domain.Session
	sessionOrder []string
	maxSessions  int
}

// NewMemoryStore initializes a MemoryStore with configured capacity.
func NewMemoryStore(maxTraces, maxSessions int) *MemoryStore {
	store := &MemoryStore{
		traceIndex:   make(map[string]*domain.Trace, maxTraces),
		spanIndex:    make(map[string]*domain.Span, maxTraces*4),
		sessions:     make(map[string]*domain.Session, maxSessions),
		sessionOrder: make([]string, 0, maxSessions),
		maxSessions:  maxSessions,
	}

	store.traces = NewRingBuffer[*domain.Trace](maxTraces, func(evicted *domain.Trace) {
		if evicted != nil {
			delete(store.traceIndex, evicted.ID)
			for _, s := range evicted.Spans {
				delete(store.spanIndex, s.ID)
			}
		}
	})

	return store
}

// SaveTrace indexes and appends a trace to the bounded buffer.
func (s *MemoryStore) SaveTrace(trace *domain.Trace) error {
	if trace == nil || trace.ID == "" {
		return ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.traceIndex[trace.ID] = trace
	for _, span := range trace.Spans {
		s.spanIndex[span.ID] = span
	}
	s.traces.Push(trace)

	if trace.SessionID != "" {
		s.updateSessionUnderLock(trace)
	}

	return nil
}

// GetTrace retrieves a trace by ID.
func (s *MemoryStore) GetTrace(id string) (*domain.Trace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trace, exists := s.traceIndex[id]
	if !exists {
		return nil, ErrNotFound
	}
	return trace, nil
}

// SaveSpan adds or updates a span in the index.
func (s *MemoryStore) SaveSpan(span *domain.Span) error {
	if span == nil || span.ID == "" {
		return ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.spanIndex[span.ID] = span
	if trace, exists := s.traceIndex[span.TraceID]; exists {
		found := false
		for i, existing := range trace.Spans {
			if existing.ID == span.ID {
				trace.Spans[i] = span
				found = true
				break
			}
		}
		if !found {
			trace.Spans = append(trace.Spans, span)
		}
	}
	return nil
}

// GetSpan retrieves a span by its ID.
func (s *MemoryStore) GetSpan(id string) (*domain.Span, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	span, exists := s.spanIndex[id]
	if !exists {
		return nil, ErrNotFound
	}
	return span, nil
}

// Count returns the total number of traces stored.
func (s *MemoryStore) Count() int {
	return s.traces.Size()
}

// Clear flushes all traces, spans, and sessions.
func (s *MemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.traces.Clear()
	clear(s.traceIndex)
	clear(s.spanIndex)
	clear(s.sessions)
	s.sessionOrder = s.sessionOrder[:0]
	return nil
}

