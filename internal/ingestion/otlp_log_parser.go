package ingestion

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ParseOTLPLogs converts OpenTelemetry OTLP/HTTP log records into domain Traces.
func ParseOTLPLogs(payload *OTLPLogsPayload) []*domain.Trace {
	if payload == nil {
		return nil
	}

	traceMap := make(map[string]*domain.Trace)

	for _, rl := range payload.ResourceLogs {
		resAttrs := flattenAttributes(rl.Resource.Attributes)
		project := resAttrs["service.name"]
		if project == "" {
			project = "claude-code"
		}

		for _, sl := range rl.ScopeLogs {
			for _, rec := range sl.LogRecords {
				span := convertLogRecordToSpan(rec, resAttrs)
				if span == nil {
					continue
				}

				traceID := span.TraceID
				if traceID == "" {
					traceID = "trc_" + span.SessionID
				}

				trace, exists := traceMap[traceID]
				if !exists {
					traceName := "Claude Code Turn"
					if span.Model != "" {
						traceName = fmt.Sprintf("Claude Code (%s)", span.Model)
					}
					trace = &domain.Trace{
						ID:        traceID,
						SessionID: span.SessionID,
						Project:   project,
						Name:      traceName,
						StartTime: span.StartTime,
						EndTime:   span.EndTime,
						Spans:     make([]*domain.Span, 0, 4),
					}
					traceMap[traceID] = trace
				}

				trace.Spans = append(trace.Spans, span)
				if trace.UserPrompt == "" {
					trace.UserPrompt = extractPromptFromAttrs(span.Metadata)
					if trace.UserPrompt != "" {
						trace.Name = trace.UserPrompt
					}
				}
				if span.StartTime.Before(trace.StartTime) || trace.StartTime.IsZero() {
					trace.StartTime = span.StartTime
				}
				if span.EndTime.After(trace.EndTime) {
					trace.EndTime = span.EndTime
				}
				if trace.SessionID == "" && span.SessionID != "" {
					trace.SessionID = span.SessionID
				}
				if (trace.Name == "" || trace.Name == "Claude Code Turn") && span.Model != "" {
					trace.Name = fmt.Sprintf("Claude Code (%s)", span.Model)
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

func convertLogRecordToSpan(rec OTLPLogRecord, resAttrs map[string]string) *domain.Span {
	attrs := flattenAttributes(rec.Attributes)
	bodyStr := rec.Body.StringValue

	eventName := attrs["event.name"]
	if eventName == "" {
		eventName = strings.TrimPrefix(bodyStr, "claude_code.")
	}
	if eventName == "user_prompt" && attrs["prompt"] == "" && bodyStr != "" && !strings.HasPrefix(bodyStr, "claude_code.") {
		attrs["prompt"] = bodyStr
	}

	sessID := attrs["session.id"]
	if sessID == "" {
		sessID = resAttrs["session.id"]
	}

	traceID := attrs["prompt.id"]
	if traceID == "" {
		traceID = rec.TraceID
	}
	if traceID == "" {
		traceID = attrs["request_id"]
	}

	spanID := rec.SpanID
	if spanID == "" {
		spanID = attrs["request_id"]
	}
	if spanID == "" {
		spanID = fmt.Sprintf("span_%s_%s", eventName, attrs["event.sequence"])
	}

	endTime := parseNanoTime(rec.TimeUnixNano)
	durMs, _ := strconv.Atoi(attrs["duration_ms"])
	startTime := endTime
	if durMs > 0 {
		startTime = endTime.Add(-time.Duration(durMs) * time.Millisecond)
	}

	kind := domain.SpanKindWorkflow
	spanName := fmt.Sprintf("Event: %s", eventName)
	model := attrs["model"]

	if strings.Contains(eventName, "api_request") || eventName == "generation" {
		kind = domain.SpanKindGeneration
		if model != "" {
			spanName = fmt.Sprintf("LLM Generation (%s)", model)
		} else {
			spanName = "LLM Generation"
		}
	} else if strings.Contains(eventName, "tool") {
		kind = domain.SpanKindTool
		toolName := attrs["tool_name"]
		if toolName == "" {
			toolName = attrs["tool"]
		}
		if toolName != "" {
			spanName = fmt.Sprintf("Tool: %s", toolName)
		} else {
			spanName = "Tool Execution"
		}
	} else if strings.Contains(eventName, "assistant_response") {
		kind = domain.SpanKindWorkflow
		spanName = "Assistant Turn"
	}

	tokens := extractOTLPUsage(attrs)
	enrichContextTokens(&tokens, attrs)

	var costUSD float64
	if costStr := attrs["cost_usd"]; costStr != "" {
		costUSD, _ = strconv.ParseFloat(costStr, 64)
	}

	ttftMs, _ := strconv.ParseFloat(attrs["ttft_ms"], 64)
	var tokensPerSec float64
	if kind == domain.SpanKindGeneration {
		tokensPerSec = computeGenVelocity(durMs, ttftMs, tokens.OutputTokens)
	}

	var tool *domain.ToolCall
	if kind == domain.SpanKindTool {
		tool = buildToolCall(attrs)
	}

	return &domain.Span{
		ID:           spanID,
		TraceID:      traceID,
		SessionID:    sessID,
		Name:         spanName,
		Kind:         kind,
		Status:       domain.StatusSuccess,
		Model:        model,
		StartTime:    startTime,
		EndTime:      endTime,
		DurationMs:   float64(durMs),
		TTFTMs:       ttftMs,
		TokensPerSec: tokensPerSec,
		Effort:       attrs["effort"],
		Speed:        attrs["speed"],
		Tokens:       tokens,
		CostUSD:      costUSD,
		Tool:         tool,
		Metadata:     attrs,
	}
}
