package ingestion_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
)

func TestParseOTLPLogs_FullWorkflow(t *testing.T) {
	payload := &ingestion.OTLPLogsPayload{
		ResourceLogs: []ingestion.OTLPResourceLog{
			{
				Resource: ingestion.OTLPResource{
					Attributes: []ingestion.OTLPAttribute{
						{Key: "service.name", Value: ingestion.OTLPValue{StringValue: "claude-code"}},
					},
				},
				ScopeLogs: []ingestion.OTLPScopeLog{
					{
						LogRecords: []ingestion.OTLPLogRecord{
							{
								TimeUnixNano: "1793395580801000000",
								Body:         ingestion.OTLPValue{StringValue: "claude_code.api_request"},
								Attributes: []ingestion.OTLPAttribute{
									{Key: "session.id", Value: ingestion.OTLPValue{StringValue: "sess_otlp_01"}},
									{Key: "prompt.id", Value: ingestion.OTLPValue{StringValue: "prompt_otlp_01"}},
									{Key: "event.name", Value: ingestion.OTLPValue{StringValue: "api_request"}},
									{Key: "model", Value: ingestion.OTLPValue{StringValue: "claude-haiku-5-5"}},
									{Key: "input_tokens", Value: ingestion.OTLPValue{IntValue: 10}},
									{Key: "output_tokens", Value: ingestion.OTLPValue{IntValue: 200}},
									{Key: "cache_read_tokens", Value: ingestion.OTLPValue{IntValue: 1500}},
									{Key: "duration_ms", Value: ingestion.OTLPValue{IntValue: 800}},
									{Key: "cost_usd", Value: ingestion.OTLPValue{DoubleValue: 0.0004}},
								},
							},
							{
								TimeUnixNano: "1793395581000000000",
								Body:         ingestion.OTLPValue{StringValue: "claude_code.tool_use"},
								Attributes: []ingestion.OTLPAttribute{
									{Key: "session.id", Value: ingestion.OTLPValue{StringValue: "sess_otlp_01"}},
									{Key: "prompt.id", Value: ingestion.OTLPValue{StringValue: "prompt_otlp_01"}},
									{Key: "event.name", Value: ingestion.OTLPValue{StringValue: "tool_use"}},
									{Key: "tool_name", Value: ingestion.OTLPValue{StringValue: "bash"}},
									{Key: "tool_input", Value: ingestion.OTLPValue{StringValue: `{"command":"ls"}`}},
									{Key: "duration_ms", Value: ingestion.OTLPValue{IntValue: 50}},
								},
							},
						},
					},
				},
			},
		},
	}

	traces := ingestion.ParseOTLPLogs(payload)
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	trc := traces[0]
	if trc.ID != "prompt_otlp_01" {
		t.Errorf("expected trace ID prompt_otlp_01, got %s", trc.ID)
	}
	if trc.SessionID != "sess_otlp_01" {
		t.Errorf("expected session ID sess_otlp_01, got %s", trc.SessionID)
	}
	if len(trc.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(trc.Spans))
	}

	genSpan := trc.Spans[0]
	if genSpan.Kind != domain.SpanKindGeneration {
		t.Errorf("expected generation span, got %v", genSpan.Kind)
	}
	if genSpan.Tokens.CacheReadInputTokens != 1500 {
		t.Errorf("expected 1500 cache read tokens, got %d", genSpan.Tokens.CacheReadInputTokens)
	}

	toolSpan := trc.Spans[1]
	if toolSpan.Kind != domain.SpanKindTool {
		t.Errorf("expected tool span, got %v", toolSpan.Kind)
	}
	if toolSpan.Tool == nil || toolSpan.Tool.Name != "bash" {
		t.Errorf("expected bash tool, got %+v", toolSpan.Tool)
	}
}

func TestParseOTLPLogs_NilAndEmpty(t *testing.T) {
	if traces := ingestion.ParseOTLPLogs(nil); traces != nil {
		t.Errorf("expected nil for nil payload, got %v", traces)
	}

	// Test assistant_response, fallback trace ID, and duration 0
	extraPayload := &ingestion.OTLPLogsPayload{
		ResourceLogs: []ingestion.OTLPResourceLog{
			{
				ScopeLogs: []ingestion.OTLPScopeLog{
					{
						LogRecords: []ingestion.OTLPLogRecord{
							{
								TimeUnixNano: "1793395580000000000",
								Body:         ingestion.OTLPValue{StringValue: "claude_code.assistant_response"},
								Attributes: []ingestion.OTLPAttribute{
									{Key: "session.id", Value: ingestion.OTLPValue{StringValue: "sess_fallback"}},
									{Key: "request_id", Value: ingestion.OTLPValue{StringValue: "req_fallback_01"}},
									{Key: "event.name", Value: ingestion.OTLPValue{StringValue: "assistant_response"}},
									{Key: "tool", Value: ingestion.OTLPValue{StringValue: "view"}},
								},
							},
						},
					},
				},
			},
		},
	}

	traces := ingestion.ParseOTLPLogs(extraPayload)
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].ID != "req_fallback_01" {
		t.Errorf("expected fallback trace ID req_fallback_01, got %s", traces[0].ID)
	}
}

