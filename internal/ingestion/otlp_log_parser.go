package ingestion

import (
	"fmt"

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
				if trace.UserOutput == "" {
					trace.UserOutput = extractResponseFromAttrs(span.Metadata)
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
