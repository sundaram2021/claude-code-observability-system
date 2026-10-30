package service_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestTraceService_IngestAndHierarchy(t *testing.T) {
	repo := repository.NewMemoryStore(10, 10)
	bus := service.NewEventBus()
	svc := service.NewTraceService(repo, bus)

	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	trace := &domain.Trace{
		ID:        "trc_tree",
		SessionID: "sess_tree",
		Name:      "Agent Prompt",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(500 * time.Millisecond),
		Spans: []*domain.Span{
			{
				ID:        "root_1",
				TraceID:   "trc_tree",
				Name:      "agent.turn",
				Kind:      domain.SpanKindWorkflow,
				StartTime: time.Now(),
				EndTime:   time.Now().Add(500 * time.Millisecond),
			},
			{
				ID:        "gen_1",
				TraceID:   "trc_tree",
				ParentID:  "root_1",
				Name:      "claude.generation",
				Kind:      domain.SpanKindGeneration,
				Model:     "claude-3-7-sonnet",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(300 * time.Millisecond),
				Tokens: domain.TokenUsage{
					InputTokens:  1000,
					OutputTokens: 200,
				},
			},
			{
				ID:        "tool_1",
				TraceID:   "trc_tree",
				ParentID:  "root_1",
				Name:      "tool:bash",
				Kind:      domain.SpanKindTool,
				StartTime: time.Now().Add(305 * time.Millisecond),
				EndTime:   time.Now().Add(450 * time.Millisecond),
				Tool: &domain.ToolCall{
					Name:  "bash",
					Input: `{"command":"go test"}`,
				},
			},
		},
	}

	if err := svc.IngestTrace(trace); err != nil {
		t.Fatalf("failed to ingest trace: %v", err)
	}

	// Verify EventBus published event
	select {
	case ev := <-sub:
		if ev.Type != service.EventTraceCreated || ev.Trace.ID != "trc_tree" {
			t.Errorf("unexpected event received: %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for event bus publication")
	}

	// Retrieve trace with hierarchy
	fetched, err := svc.GetTrace("trc_tree")
	if err != nil {
		t.Fatalf("failed to get trace: %v", err)
	}

	if fetched.RootSpan == nil {
		t.Fatalf("expected RootSpan to be constructed")
	}
	if len(fetched.RootSpan.Children) != 2 {
		t.Fatalf("expected 2 children under root span, got %d", len(fetched.RootSpan.Children))
	}
	if fetched.TotalCost <= 0 {
		t.Errorf("expected computed total cost > 0, got %f", fetched.TotalCost)
	}
}

func TestTraceService_IngestMerge(t *testing.T) {
	repo := repository.NewMemoryStore(10, 10)
	svc := service.NewTraceService(repo, nil)

	trc1 := &domain.Trace{
		ID:        "merge_trc_01",
		SessionID: "sess_merge",
		Name:      "Claude Code Turn",
		Spans: []*domain.Span{
			{ID: "span_1", TraceID: "merge_trc_01", Name: "Span 1"},
		},
	}
	_ = svc.IngestTrace(trc1)

	trc2 := &domain.Trace{
		ID:        "merge_trc_01",
		Name:      "Claude Code (claude-haiku-5-5)",
		Spans: []*domain.Span{
			{ID: "span_2", TraceID: "merge_trc_01", Name: "Span 2"},
		},
	}
	_ = svc.IngestTrace(trc2)

	fetched, _ := svc.GetTrace("merge_trc_01")
	if len(fetched.Spans) != 2 {
		t.Fatalf("expected 2 merged spans, got %d", len(fetched.Spans))
	}
	if fetched.Name != "Claude Code (claude-haiku-5-5)" {
		t.Errorf("expected updated name, got %s", fetched.Name)
	}
}


