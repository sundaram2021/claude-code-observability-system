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

	// Ingest trace with zero times, empty ID, and empty status to hit default branches
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

	// ListTraces & GetTrace
	traces, count, err := traceSvc.ListTraces(domain.TraceFilter{Limit: 10})
	if err != nil || count == 0 || len(traces) == 0 {
		t.Errorf("ListTraces failed: %v", err)
	}
	fetched, err := traceSvc.GetTrace(emptyTrace.ID)
	if err != nil || fetched == nil {
		t.Errorf("GetTrace failed: %v", err)
	}
	_, _ = traceSvc.GetTrace("nonexistent")

	// SessionService ListSessions
	_, _, err = sessSvc.ListSessions(0, 0)
	if err != nil {
		t.Errorf("unexpected error for limit 0: %v", err)
	}

	// Metrics
	metrics, err := metricsSvc.GetMetrics()
	if err != nil || metrics.ErrorCount == 0 {
		t.Errorf("expected error count > 0")
	}
}

func TestService_EnrichTracePrompt(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	traceSvc := service.NewTraceService(store, nil)

	// 1. Trace 1 in session has no prompt, generic name
	t1 := &domain.Trace{
		ID:        "trc_sess_1",
		SessionID: "sess_p1",
		Name:      "claude_code.llm_request",
		Spans: []*domain.Span{
			{ID: "sp_1", Name: "claude_code.llm_request", Kind: domain.SpanKindGeneration, DurationMs: 100},
		},
	}
	_ = traceSvc.IngestTrace(t1)

	// 2. Trace 2 in same session arrives with user prompt in span metadata
	t2 := &domain.Trace{
		ID:        "trc_sess_2",
		SessionID: "sess_p1",
		Name:      "Claude Code Turn",
		Spans: []*domain.Span{
			{
				ID:   "sp_2",
				Name: "Event: user_prompt",
				Metadata: map[string]string{
					"prompt":   "explain code",
					"response": "code summary output",
				},
			},
		},
	}
	_ = traceSvc.IngestTrace(t2)

	// Verify t2 merged into t1 into a single canonical trace
	reloadedT1, err := traceSvc.GetTrace("trc_sess_1")
	if err != nil || reloadedT1.UserPrompt != "explain code" || reloadedT1.UserOutput != "code summary output" {
		t.Errorf("expected t1 to have merged prompt and output, got %+v", reloadedT1)
	}
	if len(reloadedT1.Spans) != 2 {
		t.Errorf("expected 2 merged spans in t1, got %d", len(reloadedT1.Spans))
	}
}
