package service_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestMetricsService_Aggregations(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	metricsSvc := service.NewMetricsService(store, store)

	// Initially empty metrics
	emptyMetrics, err := metricsSvc.GetMetrics()
	if err != nil || emptyMetrics.TotalTraces != 0 {
		t.Fatalf("expected 0 total traces for empty store")
	}

	// Add traces with tool executions and models
	t1 := &domain.Trace{
		ID:        "t1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(200 * time.Millisecond),
		Spans: []*domain.Span{
			{
				ID:        "s1",
				Name:      "claude.gen",
				Kind:      domain.SpanKindGeneration,
				Model:     "claude-3-7-sonnet",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(100 * time.Millisecond),
				Tokens:    domain.TokenUsage{InputTokens: 500, OutputTokens: 100},
			},
			{
				ID:        "s2",
				Name:      "tool:bash",
				Kind:      domain.SpanKindTool,
				StartTime: time.Now(),
				EndTime:   time.Now().Add(80 * time.Millisecond),
				Tool:      &domain.ToolCall{Name: "bash", Input: "go test", Output: "PASS"},
			},
		},
	}
	_ = traceSvc.IngestTrace(t1)

	metrics, err := metricsSvc.GetMetrics()
	if err != nil {
		t.Fatalf("unexpected error getting metrics: %v", err)
	}

	if metrics.TotalTraces != 1 || metrics.TotalSpans != 2 {
		t.Errorf("expected 1 trace and 2 spans, got %d, %d", metrics.TotalTraces, metrics.TotalSpans)
	}
	if len(metrics.Tools) != 1 || metrics.Tools[0].Name != "bash" {
		t.Errorf("expected tool 'bash' in metrics: %+v", metrics.Tools)
	}
	if len(metrics.Models) != 1 || metrics.Models[0].Model != "claude-3-7-sonnet" {
		t.Errorf("expected model 'claude-3-7-sonnet' in metrics: %+v", metrics.Models)
	}
}

func TestSessionService_DetailAndPagination(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	traceSvc := service.NewTraceService(store, bus)
	sessSvc := service.NewSessionService(store, store)

	t1 := &domain.Trace{
		ID:        "trc_sess_1",
		SessionID: "sess_alpha",
		Name:      "Turn 1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
	}
	_ = traceSvc.IngestTrace(t1)

	summaries, total, err := sessSvc.ListSessions(10, 0)
	if err != nil || total != 1 || len(summaries) != 1 {
		t.Fatalf("expected 1 session summary, got %d", total)
	}

	sess, traces, err := sessSvc.GetSessionDetail("sess_alpha")
	if err != nil || sess.ID != "sess_alpha" || len(traces) != 1 {
		t.Errorf("unexpected session detail: %+v, traces: %d", sess, len(traces))
	}

	// Missing session
	_, _, err = sessSvc.GetSessionDetail("missing_sess")
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for missing session")
	}
}

func TestEventBus_UnsubscribeAndOverflow(t *testing.T) {
	bus := service.NewEventBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(ch1)
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	// Fill buffer of ch2 to trigger default non-blocking branch in Publish
	for i := 0; i < 40; i++ {
		bus.Publish(service.RealtimeEvent{Type: service.EventTraceCreated})
	}

	bus.Unsubscribe(ch2)
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestTreeBuilder_OrphanAndEmptySpans(t *testing.T) {
	// Empty slice
	if root := service.BuildSpanTree(nil); root != nil {
		t.Errorf("expected nil for empty spans")
	}

	// Spans with orphan parent
	spans := []*domain.Span{
		{ID: "orphan_1", ParentID: "non_existent_parent", Name: "orphan"},
	}
	root := service.BuildSpanTree(spans)
	if root == nil || root.ID != "orphan_1" {
		t.Errorf("expected fallback root for orphan span")
	}
}

