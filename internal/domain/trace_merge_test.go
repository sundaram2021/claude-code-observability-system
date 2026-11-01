package domain_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

func TestTrace_IsStartupPlumbing(t *testing.T) {
	var nilTrace *domain.Trace
	if nilTrace.IsStartupPlumbing() {
		t.Errorf("nil trace should not be plumbing")
	}

	emptyTrace := &domain.Trace{ID: "t1"}
	if emptyTrace.IsStartupPlumbing() {
		t.Errorf("empty spans should not be plumbing")
	}

	promptTrace := &domain.Trace{ID: "t2", UserPrompt: "hello"}
	if promptTrace.IsStartupPlumbing() {
		t.Errorf("prompt trace should not be plumbing")
	}

	outputTrace := &domain.Trace{ID: "t3", UserOutput: "world"}
	if outputTrace.IsStartupPlumbing() {
		t.Errorf("output trace should not be plumbing")
	}

	hookTrace := &domain.Trace{
		ID: "t4",
		Spans: []*domain.Span{
			{ID: "s1", Name: "Event: hook_registered", Kind: domain.SpanKindWorkflow},
			{ID: "s2", Name: "Event: managed_settings_resolved", Kind: domain.SpanKindWorkflow},
			{ID: "s3", Name: "Event: plugin_loaded", Kind: domain.SpanKindWorkflow},
		},
	}
	if !hookTrace.IsStartupPlumbing() {
		t.Errorf("pure hook trace should be detected as startup plumbing")
	}

	activeHookTrace := &domain.Trace{
		ID: "t5",
		Spans: []*domain.Span{
			{ID: "s1", Name: "Event: hook_registered", DurationMs: 150},
		},
	}
	if activeHookTrace.IsStartupPlumbing() {
		t.Errorf("active hook trace with duration should not be plumbing")
	}

	genTrace := &domain.Trace{
		ID: "t6",
		Spans: []*domain.Span{
			{ID: "s1", Name: "Event: hook_registered", Kind: domain.SpanKindGeneration},
		},
	}
	if genTrace.IsStartupPlumbing() {
		t.Errorf("gen trace should not be plumbing")
	}
}

func TestTrace_MergeFrom(t *testing.T) {
	now := time.Now()
	t1 := &domain.Trace{
		ID:        "t1",
		Name:      "Claude Code Turn",
		StartTime: now,
		EndTime:   now.Add(time.Second),
		Spans: []*domain.Span{
			{ID: "s1", Name: "workflow", Kind: domain.SpanKindWorkflow, DurationMs: 10},
		},
	}

	t1.MergeFrom(nil)

	src := &domain.Trace{
		ID:         "src1",
		Name:       "Claude Code (claude-haiku-5-5)",
		UserPrompt: "write test",
		UserOutput: "test written",
		Status:     domain.StatusError,
		StartTime:  now.Add(-time.Second),
		EndTime:    now.Add(2 * time.Second),
		Spans: []*domain.Span{
			{
				ID:         "s1",
				Name:       "generation",
				Kind:       domain.SpanKindGeneration,
				Model:      "claude-haiku-5-5",
				DurationMs: 500,
				Status:     domain.StatusError,
				Tokens:     domain.TokenUsage{TotalTokens: 100},
				Tool:       &domain.ToolCall{Name: "bash"},
				ParentID:   "p1",
			},
			{ID: "s2", Name: "tool", Kind: domain.SpanKindTool, DurationMs: 50},
		},
	}

	t1.MergeFrom(src)

	if t1.UserPrompt != "write test" || t1.UserOutput != "test written" || t1.Name != "write test" {
		t.Errorf("expected prompt/output/name updated")
	}
	if t1.Status != domain.StatusError {
		t.Errorf("expected error status merged")
	}
	if len(t1.Spans) != 2 {
		t.Fatalf("expected 2 spans merged, got %d", len(t1.Spans))
	}
	s1 := t1.Spans[0]
	if s1.Kind != domain.SpanKindGeneration || s1.Model != "claude-haiku-5-5" || s1.DurationMs != 500 || s1.Status != domain.StatusError || s1.ParentID != "p1" || s1.Tool == nil {
		t.Errorf("expected s1 enriched with richer span data, got %+v", s1)
	}
}
