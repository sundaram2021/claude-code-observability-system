package service

import (
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

// TraceService orchestrates trace and span storage, hierarchy, and notifications.
type TraceService struct {
	repo repository.TraceRepository
	bus  *EventBus
}

// NewTraceService creates an instance of TraceService.
func NewTraceService(repo repository.TraceRepository, bus *EventBus) *TraceService {
	return &TraceService{
		repo: repo,
		bus:  bus,
	}
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
		if (existing.Name == "" || existing.Name == "Claude Code Turn") && t.Name != "" && t.Name != "Claude Code Turn" {
			existing.Name = t.Name
		}
		t = existing
	}

	s.enrichSpans(t)
	s.calculateTotals(t)
	t.RootSpan = BuildSpanTree(t.Spans)

	if err := s.repo.SaveTrace(t); err != nil {
		return err
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

// enrichSpans computes costs and durations for individual spans.
func (s *TraceService) enrichSpans(t *domain.Trace) {
	for _, span := range t.Spans {
		span.CalculateDuration()
		if span.Tokens.TotalTokens == 0 {
			span.Tokens.TotalTokens = span.Tokens.InputTokens + span.Tokens.OutputTokens +
				span.Tokens.CacheReadInputTokens + span.Tokens.CacheCreationInputTokens
		}
		if span.CostUSD == 0 && span.Model != "" {
			cost, _ := domain.CalculateCost(span.Model, span.Tokens)
			span.CostUSD = cost
		}
		if span.Status == "" {
			span.Status = domain.StatusSuccess
		}
	}
}

// calculateTotals aggregates tokens, costs, performance, and overall status for the trace.
func (s *TraceService) calculateTotals(t *domain.Trace) {
	var totalCost float64
	var usage domain.TokenUsage
	var toolDur, genDur, sumTTFT, sumVel float64
	var toolCount, genCount int
	hasError, hadError, selfCorrected := false, false, false
	primaryModel := ""

	for _, span := range t.Spans {
		totalCost += span.CostUSD
		usage.InputTokens += span.Tokens.InputTokens
		usage.OutputTokens += span.Tokens.OutputTokens
		usage.CacheReadInputTokens += span.Tokens.CacheReadInputTokens
		usage.CacheCreationInputTokens += span.Tokens.CacheCreationInputTokens
		usage.CacheCreation5mTokens += span.Tokens.CacheCreation5mTokens
		usage.CacheCreation1hTokens += span.Tokens.CacheCreation1hTokens
		usage.TotalTokens += span.Tokens.TotalTokens

		if span.Model != "" && primaryModel == "" {
			primaryModel = span.Model
		}

		if span.Kind == domain.SpanKindTool {
			toolCount++
			toolDur += span.DurationMs
		} else if span.Kind == domain.SpanKindGeneration {
			genCount++
			genDur += span.DurationMs
			if span.TTFTMs > 0 {
				sumTTFT += span.TTFTMs
			}
			if span.TokensPerSec > 0 {
				sumVel += span.TokensPerSec
			}
		}

		if span.Status == domain.StatusError {
			hasError = true
			hadError = true
		} else if hadError && span.Status == domain.StatusSuccess {
			selfCorrected = true
		}
	}

	usage.ContextTokens = usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	usage.ContextWindowLimit = 200000
	if usage.ContextTokens > 0 {
		usage.ContextSaturationPct = (float64(usage.ContextTokens) / 200000.0) * 100.0
	}

	t.TotalCost = totalCost
	t.TotalTokens = usage
	t.ToolCount = toolCount
	t.ToolDurationMs = toolDur
	t.GenerationDurationMs = genDur
	t.SelfCorrected = selfCorrected
	t.ContextSaturationPct = usage.ContextSaturationPct

	if genCount > 0 {
		t.AvgTTFTMs = sumTTFT / float64(genCount)
		t.TokensPerSec = sumVel / float64(genCount)
	}

	if primaryModel != "" {
		_, savings := domain.CalculateCost(primaryModel, usage)
		t.CacheSavingsUSD = savings
		uncached := totalCost + savings
		if totalCost > 0 && uncached > 0 {
			t.CacheMultiplier = uncached / totalCost
		}
	}

	if hasError {
		t.Status = domain.StatusError
	} else if t.Status == "" {
		t.Status = domain.StatusSuccess
	}
}

