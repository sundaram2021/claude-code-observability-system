package service_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestService_CoverageBoost(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)
	metricsSvc := service.NewMetricsService(store, store)

	// 1. Ingest trace with zero times, empty ID, and empty status to hit default branches
	emptyTrace := &domain.Trace{
		Name: "Zero Times Trace",
		Spans: []*domain.Span{
			{
				ID:     "sp_zero_1",
				Name:   "tool:bash",
				Kind:   domain.SpanKindTool,
				Model:  "claude-3-7-sonnet",
				Status: domain.StatusError,
				Tokens: domain.TokenUsage{InputTokens: 100},
				Tool:   &domain.ToolCall{Name: "bash", IsError: true},
			},
		},
	}
	if err := traceSvc.IngestTrace(emptyTrace); err != nil {
		t.Fatalf("failed to ingest empty trace: %v", err)
	}
	if emptyTrace.ID == "" || emptyTrace.Status != domain.StatusError {
		t.Errorf("expected generated ID and error status")
	}

	// 2. SessionService ListSessions with limit <= 0 to hit default limit
	summaries, _, err := sessSvc.ListSessions(0, 0)
	if err != nil {
		t.Errorf("unexpected error for limit 0: %v", err)
	}
	_ = summaries

	// 3. Metrics with error span
	metrics, err := metricsSvc.GetMetrics()
	if err != nil {
		t.Fatalf("unexpected error getting metrics: %v", err)
	}
	if metrics.ErrorCount == 0 {
		t.Errorf("expected error count > 0")
	}
}
