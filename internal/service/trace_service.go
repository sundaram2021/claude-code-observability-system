package service

import (
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
	if t == nil || t.IsStartupPlumbing() {
		return nil
	}
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

	existing, _ := s.repo.GetTrace(t.ID)
	if existing == nil && t.SessionID != "" {
		existing = s.findCorrelatedTrace(t)
	}

	if existing != nil {
		existing.MergeFrom(t)
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

func (s *TraceService) findCorrelatedTrace(t *domain.Trace) *domain.Trace {
	traces, _, err := s.repo.ListTraces(domain.TraceFilter{SessionID: t.SessionID, Limit: 50})
	if err != nil || len(traces) == 0 {
		return nil
	}
	spanIDSet := make(map[string]bool, len(t.Spans))
	for _, sp := range t.Spans {
		if sp.ID != "" {
			spanIDSet[sp.ID] = true
		}
	}
	for _, cand := range traces {
		if cand.ID == t.ID {
			continue
		}
		for _, sp := range cand.Spans {
			if spanIDSet[sp.ID] {
				return cand
			}
		}
		if t.UserPrompt != "" && cand.UserPrompt != "" && t.UserPrompt == cand.UserPrompt {
			return cand
		}
		diff := cand.StartTime.Sub(t.StartTime)
		if diff < 0 {
			diff = -diff
		}
		if diff < 15*time.Second {
			if t.UserPrompt == "" || cand.UserPrompt == "" || t.UserPrompt == cand.UserPrompt {
				return cand
			}
		}
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
