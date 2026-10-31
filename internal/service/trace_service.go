package service

import (
	"strings"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

// TraceService orchestrates trace and span storage, hierarchy, and notifications.
type TraceService struct {
	repo            repository.TraceRepository
	bus             *EventBus
	persistencePath string
}

// NewTraceService creates an instance of TraceService.
func NewTraceService(repo repository.TraceRepository, bus *EventBus) *TraceService {
	return &TraceService{
		repo: repo,
		bus:  bus,
	}
}

// SetPersistencePath sets the destination file for automatic trace persistence.
func (s *TraceService) SetPersistencePath(p string) {
	s.persistencePath = p
}

// IngestTrace enriches, builds tree, computes totals, stores, and broadcasts a trace.
func (s *TraceService) IngestTrace(t *domain.Trace) error {
	if t.ID == "" {
		t.ID = generateID("trc_")
	}
	if t.StartTime.IsZero() {
		t.StartTime = time.Now()
	}
	if t.EndTime.IsZero() {
		t.EndTime = time.Now()
	}
	if t.DurationMs == 0 {
		t.DurationMs = float64(t.EndTime.Sub(t.StartTime).Microseconds()) / 1000.0
	}

	if existing, err := s.repo.GetTrace(t.ID); err == nil && existing != nil {
		spanIDs := make(map[string]bool, len(existing.Spans))
		for _, sp := range existing.Spans {
			spanIDs[sp.ID] = true
		}
		for _, sp := range t.Spans {
			if !spanIDs[sp.ID] {
				existing.Spans = append(existing.Spans, sp)
			}
		}
		if t.EndTime.After(existing.EndTime) {
			existing.EndTime = t.EndTime
		}
		if existing.SessionID == "" && t.SessionID != "" {
			existing.SessionID = t.SessionID
		}
		if existing.UserPrompt == "" && t.UserPrompt != "" {
			existing.UserPrompt = t.UserPrompt
		}
		if t.UserPrompt == "" && existing.UserPrompt != "" {
			t.UserPrompt = existing.UserPrompt
		}
		if (existing.Name == "" || existing.Name == "Claude Code Turn" || strings.HasPrefix(existing.Name, "claude_code.")) && t.Name != "" && !strings.HasPrefix(t.Name, "claude_code.") {
			existing.Name = t.Name
		}
		t = existing
	}

	enrichTracePrompt(t, s.repo)
	enrichSpans(t)
	calculateTotals(t)
	t.RootSpan = BuildSpanTree(t.Spans)

	if err := s.repo.SaveTrace(t); err != nil {
		return err
	}
	if s.persistencePath != "" {
		_ = repository.SaveToFile(s.repo, s.persistencePath)
	}

	if s.bus != nil {
		s.bus.Publish(RealtimeEvent{
			Type:  EventTraceCreated,
			Trace: t,
		})
	}
	return nil
}

// GetTrace retrieves a trace by ID with reconstructed span hierarchy.
func (s *TraceService) GetTrace(id string) (*domain.Trace, error) {
	trace, err := s.repo.GetTrace(id)
	if err != nil {
		return nil, err
	}
	if trace.RootSpan == nil && len(trace.Spans) > 0 {
		trace.RootSpan = BuildSpanTree(trace.Spans)
	}
	return trace, nil
}

// ListTraces retrieves traces matching the provided filter.
func (s *TraceService) ListTraces(filter domain.TraceFilter) ([]*domain.Trace, int, error) {
	return s.repo.ListTraces(filter)
}
