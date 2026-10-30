package ingestion

import (
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ParseClaudeEvent converts a Claude Code event payload into a domain Trace.
func ParseClaudeEvent(e *ClaudeEventPayload) *domain.Trace {
	if e == nil {
		return nil
	}

	now := e.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	traceID := e.TurnID
	if traceID == "" {
		traceID = "turn_" + e.SessionID + "_" + now.Format("150405.000")
	}

	rootSpanID := "root_" + traceID
	status := domain.StatusSuccess
	if e.IsError {
		status = domain.StatusError
	}

	trace := &domain.Trace{
		ID:        traceID,
		SessionID: e.SessionID,
		Project:   e.Project,
		Name:      formatTurnName(e),
		Status:    status,
		StartTime: now.Add(-time.Duration(e.Duration * float64(time.Millisecond))),
		EndTime:   now,
		Spans:     make([]*domain.Span, 0, 4),
		Metadata:  e.Metadata,
	}

	rootSpan := &domain.Span{
		ID:        rootSpanID,
		TraceID:   traceID,
		SessionID: e.SessionID,
		Name:      trace.Name,
		Kind:      domain.SpanKindWorkflow,
		Status:    status,
		StartTime: trace.StartTime,
		EndTime:   trace.EndTime,
	}
	trace.Spans = append(trace.Spans, rootSpan)

	if e.Model != "" || e.Response != "" {
		trace.Spans = append(trace.Spans, buildGenerationSpan(e, traceID, rootSpanID, now))
	}

	if e.ToolName != "" {
		trace.Spans = append(trace.Spans, buildToolSpan(e, traceID, rootSpanID, now))
	}

	return trace
}

func formatTurnName(e *ClaudeEventPayload) string {
	if e.Prompt != "" {
		if len(e.Prompt) > 40 {
			return "Prompt: " + e.Prompt[:37] + "..."
		}
		return "Prompt: " + e.Prompt
	}
	if e.ToolName != "" {
		return "Tool: " + e.ToolName
	}
	return "Claude Code Turn"
}

func buildGenerationSpan(e *ClaudeEventPayload, traceID, parentID string, now time.Time) *domain.Span {
	usage := domain.TokenUsage{
		InputTokens:              e.Tokens.InputTokens,
		OutputTokens:             e.Tokens.OutputTokens,
		CacheReadInputTokens:     e.Tokens.CacheReadTokens,
		CacheCreationInputTokens: e.Tokens.CacheCreateTokens,
	}
	model := e.Model
	if model == "" {
		model = "claude-3-7-sonnet"
	}

	return &domain.Span{
		ID:        "gen_" + traceID,
		TraceID:   traceID,
		ParentID:  parentID,
		SessionID: e.SessionID,
		Name:      "claude.generation",
		Kind:      domain.SpanKindGeneration,
		Status:    domain.StatusSuccess,
		Model:     model,
		StartTime: now.Add(-time.Duration(e.Duration * float64(time.Millisecond))),
		EndTime:   now,
		Tokens:    usage,
	}
}

func buildToolSpan(e *ClaudeEventPayload, traceID, parentID string, now time.Time) *domain.Span {
	status := domain.StatusSuccess
	if e.IsError || e.ExitCode != 0 {
		status = domain.StatusError
	}

	return &domain.Span{
		ID:        "tool_" + traceID + "_" + e.ToolName,
		TraceID:   traceID,
		ParentID:  parentID,
		SessionID: e.SessionID,
		Name:      "tool:" + e.ToolName,
		Kind:      domain.SpanKindTool,
		Status:    status,
		StartTime: now.Add(-time.Duration(e.Duration * float64(time.Millisecond))),
		EndTime:   now,
		Tool: &domain.ToolCall{
			Name:     e.ToolName,
			Input:    e.ToolInput,
			Output:   e.ToolOut,
			ExitCode: e.ExitCode,
			IsError:  status == domain.StatusError,
		},
	}
}

