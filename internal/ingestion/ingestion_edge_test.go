package ingestion_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
)

func TestIngestion_EdgeCasesAndFallbacks(t *testing.T) {
	// 1. ParseClaudeEvent nil
	if tr := ingestion.ParseClaudeEvent(nil); tr != nil {
		t.Errorf("expected nil trace for nil event")
	}

	// 2. Long prompt truncation
	longPrompt := "This is an extremely long user prompt intended to test whether the turn name formatting truncates appropriately at forty characters"
	ev := &ingestion.ClaudeEventPayload{
		SessionID: "s_long",
		Prompt:    longPrompt,
		Timestamp: time.Now(),
	}
	trLong := ingestion.ParseClaudeEvent(ev)
	if len(trLong.Name) > 50 {
		t.Errorf("expected truncated name, got %s", trLong.Name)
	}

	// 3. Tool only event with error
	evToolErr := &ingestion.ClaudeEventPayload{
		SessionID: "s_err",
		ToolName:  "bash",
		ExitCode:  1,
		IsError:   true,
		Duration:  50.0,
	}
	trToolErr := ingestion.ParseClaudeEvent(evToolErr)
	if trToolErr.Status != domain.StatusError {
		t.Errorf("expected status error for tool failure")
	}

	// 4. OTLP attribute types (int as float64, int as string, bool values, nano time invalid)
	payload := &ingestion.OTLPTracesPayload{
		ResourceSpans: []ingestion.OTLPResourceSpan{
			{
				Resource: ingestion.OTLPResource{
					Attributes: []ingestion.OTLPAttribute{
						{Key: "service.namespace", Value: ingestion.OTLPValue{StringValue: "claude-corp"}},
						{Key: "is_prod", Value: ingestion.OTLPValue{BoolValue: true}},
					},
				},
				ScopeSpans: []ingestion.OTLPScopeSpan{
					{
						Spans: []ingestion.OTLPSpan{
							{
								TraceID:           "trc_edge_01",
								SpanID:            "sp_edge_01",
								Name:              "edge.span",
								StartTimeUnixNano: "invalid-nano",
								EndTimeUnixNano:   "",
								Attributes: []ingestion.OTLPAttribute{
									{Key: "tool.name", Value: ingestion.OTLPValue{StringValue: "git_diff"}},
									{Key: "gen_ai.usage.input_tokens", Value: ingestion.OTLPValue{DoubleValue: 120.0}},
									{Key: "gen_ai.usage.output_tokens", Value: ingestion.OTLPValue{IntValue: "45"}},
									{Key: "error", Value: ingestion.OTLPValue{StringValue: "true"}},
								},
								Status: ingestion.OTLPStatus{Code: 2},
							},
						},
					},
				},
			},
		},
	}

	traces := ingestion.ParseOTLPTraces(payload)
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	sp := traces[0].Spans[0]
	if sp.Status != domain.StatusError {
		t.Errorf("expected status error for OTLP error code")
	}
	if sp.Tool == nil || sp.Tool.Name != "git_diff" {
		t.Errorf("expected tool git_diff from fallback tool.name attribute")
	}
}

