package ingestion_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
)

func TestParseOTLPTraces(t *testing.T) {
	payload := &ingestion.OTLPTracesPayload{
		ResourceSpans: []ingestion.OTLPResourceSpan{
			{
				Resource: ingestion.OTLPResource{
					Attributes: []ingestion.OTLPAttribute{
						{Key: "service.name", Value: ingestion.OTLPValue{StringValue: "claude-code-cli"}},
						{Key: "session.id", Value: ingestion.OTLPValue{StringValue: "sess_otlp_1"}},
					},
				},
				ScopeSpans: []ingestion.OTLPScopeSpan{
					{
						Spans: []ingestion.OTLPSpan{
							{
								TraceID:           "otlp_trc_001",
								SpanID:            "sp_01",
								Name:              "tool:bash",
								StartTimeUnixNano: "1700000000000000000",
								EndTimeUnixNano:   "1700000000500000000",
								Attributes: []ingestion.OTLPAttribute{
									{Key: "gen_ai.tool.name", Value: ingestion.OTLPValue{StringValue: "bash"}},
									{Key: "gen_ai.tool.arguments", Value: ingestion.OTLPValue{StringValue: `{"command":"npm test"}`}},
									{Key: "gen_ai.tool.result", Value: ingestion.OTLPValue{StringValue: "Tests passed"}},
								},
							},
							{
								TraceID:           "otlp_trc_001",
								SpanID:            "sp_02",
								ParentSpanID:      "sp_01",
								Name:              "claude.generation",
								StartTimeUnixNano: "1700000000000000000",
								EndTimeUnixNano:   "1700000000400000000",
								Attributes: []ingestion.OTLPAttribute{
									{Key: "gen_ai.request.model", Value: ingestion.OTLPValue{StringValue: "claude-3-5-sonnet"}},
									{Key: "gen_ai.usage.input_tokens", Value: ingestion.OTLPValue{IntValue: 800}},
									{Key: "gen_ai.usage.output_tokens", Value: ingestion.OTLPValue{IntValue: 250}},
									{Key: "gen_ai.usage.cache_read_input_tokens", Value: ingestion.OTLPValue{IntValue: 2000}},
								},
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

	trc := traces[0]
	if trc.ID != "otlp_trc_001" {
		t.Errorf("expected trace id otlp_trc_001, got %s", trc.ID)
	}
	if trc.Project != "claude-code-cli" {
		t.Errorf("expected project claude-code-cli, got %s", trc.Project)
	}
	if len(trc.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(trc.Spans))
	}

	// Verify tool span
	var toolSpan *domain.Span
	var genSpan *domain.Span
	for _, s := range trc.Spans {
		if s.Kind == domain.SpanKindTool {
			toolSpan = s
		} else if s.Kind == domain.SpanKindGeneration {
			genSpan = s
		}
	}

	if toolSpan == nil || toolSpan.Tool == nil || toolSpan.Tool.Name != "bash" {
		t.Fatalf("expected tool span with name 'bash'")
	}
	if genSpan == nil || genSpan.Tokens.InputTokens != 800 || genSpan.Tokens.CacheReadInputTokens != 2000 {
		t.Fatalf("expected generation span with correct token counts")
	}
}

