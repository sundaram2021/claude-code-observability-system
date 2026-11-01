package ingestion

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// buildToolCall extracts and formats tool execution details from attributes.
func buildToolCall(attrs map[string]string) *domain.ToolCall {
	name := attrs["tool_name"]
	if name == "" {
		name = attrs["tool"]
	}
	if name == "" {
		return nil
	}
	exitCode, _ := strconv.Atoi(attrs["tool_exit_code"])
	return &domain.ToolCall{
		Name:     name,
		Input:    attrs["tool_input"],
		Output:   attrs["tool_result"],
		ExitCode: exitCode,
		IsError:  attrs["is_error"] == "true" || exitCode != 0,
	}
}

// computeGenVelocity calculates streaming tokens per second.
func computeGenVelocity(durMs int, ttftMs float64, outTokens int) float64 {
	if durMs <= 0 || outTokens <= 0 {
		return 0
	}
	sec := (float64(durMs) - ttftMs) / 1000.0
	if sec <= 0.05 {
		sec = float64(durMs) / 1000.0
	}
	if sec > 0 {
		return float64(outTokens) / sec
	}
	return 0
}

// enrichContextTokens computes total context tokens and saturation percentage.
func enrichContextTokens(u *domain.TokenUsage, attrs map[string]string) {
	u.CacheCreation5mTokens, _ = strconv.Atoi(attrs["cache_creation_5m_tokens"])
	u.CacheCreation1hTokens, _ = strconv.Atoi(attrs["cache_creation_1h_tokens"])
	u.ContextTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
	u.ContextWindowLimit = 200000
	if u.ContextTokens > 0 {
		u.ContextSaturationPct = (float64(u.ContextTokens) / 200000.0) * 100.0
	}
}

// extractPromptFromAttrs extracts user prompt from known metadata keys.
func extractPromptFromAttrs(attrs map[string]string) string {
	if attrs == nil {
		return ""
	}
	for _, key := range []string{"user_prompt", "prompt", "gen_ai.prompt", "query"} {
		if val := attrs[key]; val != "" {
			return val
		}
	}
	return ""
}

// extractResponseFromAttrs extracts assistant response/output from metadata.
func extractResponseFromAttrs(attrs map[string]string) string {
	if attrs == nil {
		return ""
	}
	for _, key := range []string{"response", "assistant_response", "text", "content", "gen_ai.completion"} {
		if val := attrs[key]; val != "" && !strings.HasPrefix(val, "claude_code.") {
			return val
		}
	}
	return ""
}

// parseNanoTime converts unix nanosecond string into time.Time.
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

// convertLogRecordToSpan translates an OTLP log record to a domain Span.
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
		spanName = "LLM Generation"
		if model != "" {
			spanName = fmt.Sprintf("LLM Generation (%s)", model)
		}
	} else if strings.Contains(eventName, "tool") {
		kind = domain.SpanKindTool
		tName := attrs["tool_name"]
		if tName == "" {
			tName = attrs["tool"]
		}
		spanName = "Tool Execution"
		if tName != "" {
			spanName = fmt.Sprintf("Tool: %s", tName)
		}
	} else if strings.Contains(eventName, "assistant_response") {
		kind = domain.SpanKindWorkflow
		spanName = "Assistant Turn"
	}

	tokens := extractOTLPUsage(attrs)
	enrichContextTokens(&tokens, attrs)
	costUSD, _ := strconv.ParseFloat(attrs["cost_usd"], 64)
	ttftMs, _ := strconv.ParseFloat(attrs["ttft_ms"], 64)
	var tokensPerSec float64
	if kind == domain.SpanKindGeneration {
		tokensPerSec = computeGenVelocity(durMs, ttftMs, tokens.OutputTokens)
	}

	return &domain.Span{
		ID: spanID, TraceID: traceID, SessionID: sessID, Name: spanName,
		Kind: kind, Status: domain.StatusSuccess, Model: model,
		StartTime: startTime, EndTime: endTime, DurationMs: float64(durMs),
		TTFTMs: ttftMs, TokensPerSec: tokensPerSec, Effort: attrs["effort"],
		Speed: attrs["speed"], Tokens: tokens, CostUSD: costUSD,
		Tool: buildToolCall(attrs), Metadata: attrs,
	}
}
