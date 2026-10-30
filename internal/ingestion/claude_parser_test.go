package ingestion_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
)

func TestParseClaudeEvent(t *testing.T) {
	now := time.Now()
	payload := &ingestion.ClaudeEventPayload{
		SessionID: "sess_claude_101",
		Project:   "my-web-app",
		TurnID:    "turn_01",
		Prompt:    "Fix bug in router",
		Model:     "claude-3-7-sonnet",
		ToolName:  "file_edit",
		ToolInput: `{"file_path":"router.go"}`,
		ToolOut:   "File edited successfully",
		ExitCode:  0,
		Duration:  250.0,
		Tokens: ingestion.ClaudeTokenUsage{
			InputTokens:       500,
			OutputTokens:      120,
			CacheReadTokens:   4000,
			CacheCreateTokens: 1000,
		},
		Timestamp: now,
	}

	trace := ingestion.ParseClaudeEvent(payload)
	if trace == nil {
		t.Fatalf("expected parsed trace, got nil")
	}

	if trace.ID != "turn_01" {
		t.Errorf("expected trace id turn_01, got %s", trace.ID)
	}
	if trace.SessionID != "sess_claude_101" {
		t.Errorf("expected session id sess_claude_101, got %s", trace.SessionID)
	}
	if trace.Project != "my-web-app" {
		t.Errorf("expected project my-web-app, got %s", trace.Project)
	}

	// Should produce root span, generation span, and tool span
	if len(trace.Spans) != 3 {
		t.Fatalf("expected 3 spans, got %d", len(trace.Spans))
	}

	var hasTool, hasGen, hasRoot bool
	for _, s := range trace.Spans {
		switch s.Kind {
		case domain.SpanKindWorkflow:
			hasRoot = true
		case domain.SpanKindGeneration:
			hasGen = true
			if s.Tokens.CacheReadInputTokens != 4000 {
				t.Errorf("expected 4000 cache read tokens, got %d", s.Tokens.CacheReadInputTokens)
			}
		case domain.SpanKindTool:
			hasTool = true
			if s.Tool == nil || s.Tool.Name != "file_edit" {
				t.Errorf("expected tool file_edit, got %+v", s.Tool)
			}
		}
	}

	if !hasRoot || !hasGen || !hasTool {
		t.Errorf("missing expected span kinds: root=%v, gen=%v, tool=%v", hasRoot, hasGen, hasTool)
	}
}

