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

func TestParseOTLPTraces_ClaudeCodeFormat(t *testing.T) {
	payload := &ingestion.OTLPTracesPayload{
		ResourceSpans: []ingestion.OTLPResourceSpan{
			{
				Resource: ingestion.OTLPResource{
					Attributes: []ingestion.OTLPAttribute{
						{Key: "service.name", Value: ingestion.OTLPValue{StringValue: "claude-code"}},
					},
				},
				ScopeSpans: []ingestion.OTLPScopeSpan{
					{
						Spans: []ingestion.OTLPSpan{
							{
								TraceID:           "744a3ff84a5240eed20fc9eaddd0077d",
								SpanID:            "64f5c4e411efdaeb",
								Name:              "claude_code.tool",
								StartTimeUnixNano: "1793482059038000000",
								EndTimeUnixNano:   "1793482062562644500",
								Attributes: []ingestion.OTLPAttribute{
									{Key: "tool_name", Value: ingestion.OTLPValue{StringValue: "Bash"}},
									{Key: "full_command", Value: ingestion.OTLPValue{StringValue: "git status"}},
									{Key: "session.id", Value: ingestion.OTLPValue{StringValue: "sess_claude_1"}},
								},
								Events: []ingestion.OTLPEvent{
									{
										Name: "tool.output",
										Attributes: []ingestion.OTLPAttribute{
											{Key: "output", Value: ingestion.OTLPValue{StringValue: "On branch main"}},
										},
									},
								},
							},
							{
								TraceID:           "744a3ff84a5240eed20fc9eaddd0077d",
								SpanID:            "403ceb3d00b15f6f",
								Name:              "claude_code.llm_request",
								StartTimeUnixNano: "1793482057305000000",
								EndTimeUnixNano:   "1793482058684626100",
								Attributes: []ingestion.OTLPAttribute{
									{Key: "model", Value: ingestion.OTLPValue{StringValue: "claude-haiku-5-5"}},
									{Key: "input_tokens", Value: ingestion.OTLPValue{IntValue: 15}},
									{Key: "output_tokens", Value: ingestion.OTLPValue{IntValue: 80}},
									{Key: "cache_read_tokens", Value: ingestion.OTLPValue{IntValue: 5000}},
									{Key: "ttft_ms", Value: ingestion.OTLPValue{IntValue: 450}},
									{Key: "effort", Value: ingestion.OTLPValue{StringValue: "medium"}},
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
	if len(trc.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(trc.Spans))
	}

	var toolSpan, llmSpan *domain.Span
	for _, s := range trc.Spans {
		if s.Kind == domain.SpanKindTool {
			toolSpan = s
		} else if s.Kind == domain.SpanKindGeneration {
			llmSpan = s
		}
	}

	if toolSpan == nil || toolSpan.Tool == nil || toolSpan.Tool.Name != "Bash" {
		t.Fatalf("expected Bash tool span")
	}
	if toolSpan.Tool.Input != "git status" || toolSpan.Tool.Output != "On branch main" {
		t.Errorf("tool input/output mismatch: input=%s, output=%s", toolSpan.Tool.Input, toolSpan.Tool.Output)
	}

	if llmSpan == nil || llmSpan.Model != "claude-haiku-5-5" {
		t.Fatalf("expected claude-haiku-5-5 LLM span")
	}
	if llmSpan.TTFTMs != 450 || llmSpan.Tokens.CacheReadInputTokens != 5000 {
		t.Errorf("LLM TTFT or cache tokens mismatch")
	}
}

