package ingestion

import (
	"strconv"
	"strings"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ParseOTLPTraces converts an OpenTelemetry traces payload into domain Traces.
func ParseOTLPTraces(payload *OTLPTracesPayload) []*domain.Trace {
	if payload == nil {
		return nil
	}

	traceMap := make(map[string]*domain.Trace)

	for _, rs := range payload.ResourceSpans {
		resAttrs := flattenAttributes(rs.Resource.Attributes)
		project := resAttrs["service.name"]
		if project == "" {
			project = resAttrs["service.namespace"]
		}

		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				span := convertOTLPSpan(s, resAttrs)
				trace, exists := traceMap[span.TraceID]
				if !exists {
					trace = &domain.Trace{
						ID:        span.TraceID,
						SessionID: span.SessionID,
						Project:   project,
						Name:      span.Name,
						StartTime: span.StartTime,
						EndTime:   span.EndTime,
						Spans:     make([]*domain.Span, 0, 8),
					}
					traceMap[span.TraceID] = trace
				}
				trace.Spans = append(trace.Spans, span)
				if span.StartTime.Before(trace.StartTime) || trace.StartTime.IsZero() {
					trace.StartTime = span.StartTime
				}
				if span.EndTime.After(trace.EndTime) {
					trace.EndTime = span.EndTime
				}
			}
		}
	}

	result := make([]*domain.Trace, 0, len(traceMap))
	for _, t := range traceMap {
		result = append(result, t)
	}
	return result
}

func convertOTLPSpan(s OTLPSpan, resAttrs map[string]string) *domain.Span {
	attrs := flattenAttributes(s.Attributes)
	start := parseNanoTime(s.StartTimeUnixNano)
	end := parseNanoTime(s.EndTimeUnixNano)

	status := domain.StatusSuccess
	if s.Status.Code == 2 || attrs["error"] == "true" {
		status = domain.StatusError
	}

	kind := domain.SpanKindWorkflow
	if strings.Contains(s.Name, "tool") || attrs["gen_ai.tool.name"] != "" {
		kind = domain.SpanKindTool
	} else if strings.Contains(s.Name, "generation") || attrs["gen_ai.request.model"] != "" {
		kind = domain.SpanKindGeneration
	}

	model := attrs["gen_ai.request.model"]
	if model == "" {
		model = attrs["llm.model_name"]
	}

	var tool *domain.ToolCall
	toolName := attrs["gen_ai.tool.name"]
	if toolName == "" {
		toolName = attrs["tool.name"]
	}
	if toolName != "" {
		tool = &domain.ToolCall{
			Name:    toolName,
			Input:   attrs["gen_ai.tool.arguments"],
			Output:  attrs["gen_ai.tool.result"],
			IsError: status == domain.StatusError,
		}
	}

	usage := extractOTLPUsage(attrs)
	sessID := attrs["session.id"]
	if sessID == "" {
		sessID = resAttrs["session.id"]
	}

	return &domain.Span{
		ID:        s.SpanID,
		TraceID:   s.TraceID,
		ParentID:  s.ParentSpanID,
		SessionID: sessID,
		Name:      s.Name,
		Kind:      kind,
		Status:    status,
		Model:     model,
		StartTime: start,
		EndTime:   end,
		Tokens:    usage,
		Tool:      tool,
		Metadata:  attrs,
	}
}

func flattenAttributes(attrs []OTLPAttribute) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		if a.Value.StringValue != "" {
			m[a.Key] = a.Value.StringValue
		} else if a.Value.IntValue != nil {
			m[a.Key] = strconv.Itoa(toInt(a.Value.IntValue))
		} else if a.Value.DoubleValue != 0 {
			m[a.Key] = strconv.FormatFloat(a.Value.DoubleValue, 'f', -1, 64)
		} else if a.Value.BoolValue {
			m[a.Key] = "true"
		}
	}
	return m
}

func extractOTLPUsage(attrs map[string]string) domain.TokenUsage {
	return domain.TokenUsage{
		InputTokens:              parseNum(attrs, "gen_ai.usage.input_tokens", "llm.token_count.prompt", "input_tokens"),
		OutputTokens:             parseNum(attrs, "gen_ai.usage.output_tokens", "llm.token_count.completion", "output_tokens"),
		CacheReadInputTokens:     parseNum(attrs, "gen_ai.usage.cache_read_input_tokens", "anthropic.cache_read_tokens", "cache_read_tokens"),
		CacheCreationInputTokens: parseNum(attrs, "gen_ai.usage.cache_creation_input_tokens", "anthropic.cache_creation_tokens", "cache_creation_tokens"),
	}
}

func parseNum(attrs map[string]string, keys ...string) int {
	for _, k := range keys {
		if val, ok := attrs[k]; ok {
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		}
	}
	return 0
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

func parseNanoTime(nanoStr string) time.Time {
	if nanoStr == "" {
		return time.Now()
	}
	nano, err := strconv.ParseInt(nanoStr, 10, 64)
	if err != nil {
		return time.Now()
	}
	return time.Unix(0, nano)
}

