package ingestion_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
)

func TestParseLangfuseBatch(t *testing.T) {
	// Nil and empty payload checks
	if traces := ingestion.ParseLangfuseBatch(nil); traces != nil {
		t.Errorf("expected nil for nil payload")
	}
	if traces := ingestion.ParseLangfuseBatch(&ingestion.LangfuseIngestionPayload{}); traces != nil {
		t.Errorf("expected nil for empty batch")
	}

	now := time.Now()
	payload := &ingestion.LangfuseIngestionPayload{
		Batch: []ingestion.LangfuseBatchItem{
			{
				ID:        "item_1",
				Type:      "trace-create",
				Timestamp: now,
				Body: ingestion.LangfuseObservation{
					ID:        "trc_lf_100",
					SessionID: "sess_lf_1",
					Name:      "Langfuse Test Trace",
					StartTime: now,
					EndTime:   now.Add(100 * time.Millisecond),
					Level:     "DEFAULT",
				},
			},
			{
				ID:        "item_2",
				Type:      "generation-create",
				Timestamp: now,
				Body: ingestion.LangfuseObservation{
					ID:                  "gen_lf_1",
					TraceID:             "trc_lf_100",
					ParentObservationID: "trc_lf_100",
					Name:                "claude-generation",
					Model:               "claude-3-5-sonnet",
					StartTime:           now,
					EndTime:             now.Add(80 * time.Millisecond),
					Usage: &ingestion.LangfuseUsage{
						Input:  500,
						Output: 150,
						Total:  650,
					},
				},
			},
			{
				ID:        "item_3",
				Type:      "span-create",
				Timestamp: now,
				Body: ingestion.LangfuseObservation{
					ID:                  "tool_lf_1",
					TraceID:             "trc_lf_100",
					ParentObservationID: "trc_lf_100",
					Name:                "bash",
					Level:               "ERROR",
					Input:               map[string]any{"command": "cat missing.txt"},
					Output:              "file not found",
				},
			},
		},
	}

	traces := ingestion.ParseLangfuseBatch(payload)
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	trc := traces[0]
	if trc.ID != "trc_lf_100" || trc.SessionID != "sess_lf_1" {
		t.Errorf("unexpected trace id or session: %+v", trc)
	}
	if len(trc.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(trc.Spans))
	}

	var hasToolErr, hasGen bool
	for _, s := range trc.Spans {
		if s.Kind == domain.SpanKindTool {
			if s.Status != domain.StatusError {
				t.Errorf("expected tool span status error")
			}
			if s.Tool == nil || s.Tool.Name != "bash" {
				t.Errorf("expected bash tool call")
			}
			hasToolErr = true
		} else if s.Kind == domain.SpanKindGeneration {
			if s.Tokens.InputTokens != 500 || s.Tokens.OutputTokens != 150 {
				t.Errorf("expected tokens 500/150, got %+v", s.Tokens)
			}
			hasGen = true
		}
	}

	if !hasToolErr || !hasGen {
		t.Errorf("missing tool or generation span")
	}
}

