package ingestion

import (
	"encoding/json"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ParseLangfuseBatch converts a Langfuse batch payload into domain Traces.
func ParseLangfuseBatch(payload *LangfuseIngestionPayload) []*domain.Trace {
	if payload == nil || len(payload.Batch) == 0 {
		return nil
	}

	traceMap := make(map[string]*domain.Trace)

	for _, item := range payload.Batch {
		switch item.Type {
		case "trace-create":
			t := createTraceFromLangfuse(item.Body)
			traceMap[t.ID] = t
		case "span-create", "generation-create", "event-create":
			span := createSpanFromLangfuse(item)
			trace, exists := traceMap[span.TraceID]
			if !exists {
				trace = &domain.Trace{
					ID:        span.TraceID,
					SessionID: span.SessionID,
					Name:      span.Name,
					StartTime: span.StartTime,
					EndTime:   span.EndTime,
					Spans:     make([]*domain.Span, 0, 4),
				}
				traceMap[span.TraceID] = trace
			}
			trace.Spans = append(trace.Spans, span)
		}
	}

	result := make([]*domain.Trace, 0, len(traceMap))
	for _, t := range traceMap {
		result = append(result, t)
	}
	return result
}

func createTraceFromLangfuse(body LangfuseObservation) *domain.Trace {
	return &domain.Trace{
		ID:        body.ID,
		SessionID: body.SessionID,
		Name:      body.Name,
		StartTime: body.StartTime,
		EndTime:   body.EndTime,
		Status:    parseLangfuseStatus(body.Level),
		Spans:     make([]*domain.Span, 0, 4),
		Metadata:  body.Metadata,
	}
}

func createSpanFromLangfuse(item LangfuseBatchItem) *domain.Span {
	b := item.Body
	kind := domain.SpanKindWorkflow
	if item.Type == "generation-create" {
		kind = domain.SpanKindGeneration
	} else if item.Type == "span-create" {
		kind = domain.SpanKindTool
	}

	var usage domain.TokenUsage
	if b.Usage != nil {
		usage.InputTokens = b.Usage.Input
		usage.OutputTokens = b.Usage.Output
		usage.TotalTokens = b.Usage.Total
	}

	start := b.StartTime
	if start.IsZero() {
		start = item.Timestamp
	}
	end := b.EndTime
	if end.IsZero() {
		end = time.Now()
	}

	var tool *domain.ToolCall
	if kind == domain.SpanKindTool {
		tool = &domain.ToolCall{
			Name:   b.Name,
			Input:  stringifyData(b.Input),
			Output: stringifyData(b.Output),
		}
	}

	return &domain.Span{
		ID:        b.ID,
		TraceID:   b.TraceID,
		ParentID:  b.ParentObservationID,
		SessionID: b.SessionID,
		Name:      b.Name,
		Kind:      kind,
		Status:    parseLangfuseStatus(b.Level),
		Model:     b.Model,
		StartTime: start,
		EndTime:   end,
		Tokens:    usage,
		Tool:      tool,
		Metadata:  b.Metadata,
	}
}

func parseLangfuseStatus(level string) domain.SpanStatus {
	if level == "ERROR" {
		return domain.StatusError
	}
	return domain.StatusSuccess
}

func stringifyData(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
