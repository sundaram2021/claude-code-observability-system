package handler

import (
	"net/http"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/ingestion"
	"github.com/claude-code/observability/internal/service"
)

// IngestHandler routes and processes incoming telemetry from multiple protocols.
type IngestHandler struct {
	traceService *service.TraceService
}

// NewIngestHandler creates an instance of IngestHandler.
func NewIngestHandler(traceService *service.TraceService) *IngestHandler {
	return &IngestHandler{
		traceService: traceService,
	}
}

// IngestOTLP handles POST /v1/traces (OpenTelemetry OTLP/HTTP format).
func (h *IngestHandler) IngestOTLP(w http.ResponseWriter, r *http.Request) {
	var payload ingestion.OTLPTracesPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed OTLP payload: "+err.Error())
		return
	}

	traces := ingestion.ParseOTLPTraces(&payload)
	for _, t := range traces {
		if err := h.traceService.IngestTrace(t); err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{"partialSuccess": map[string]any{}})
}

// IngestClaudeEvent handles POST /api/v1/events (Claude Code hook events).
func (h *IngestHandler) IngestClaudeEvent(w http.ResponseWriter, r *http.Request) {
	var event ingestion.ClaudeEventPayload
	if err := ReadJSON(r, &event); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed Claude event JSON: "+err.Error())
		return
	}

	trace := ingestion.ParseClaudeEvent(&event)
	if trace == nil {
		WriteError(w, http.StatusBadRequest, "empty or unparseable event")
		return
	}

	if err := h.traceService.IngestTrace(trace); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusCreated, trace, 1)
}

// IngestLangfuse handles POST /api/public/ingestion (Langfuse batch format).
func (h *IngestHandler) IngestLangfuse(w http.ResponseWriter, r *http.Request) {
	var payload ingestion.LangfuseIngestionPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed Langfuse payload: "+err.Error())
		return
	}

	traces := ingestion.ParseLangfuseBatch(&payload)
	for _, t := range traces {
		_ = h.traceService.IngestTrace(t)
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"successes": len(payload.Batch),
		"errors":    []any{},
	})
}

// GetTelemetryLegacy handles GET /api/telemetry for backward compatibility with old frontend.
func (h *IngestHandler) GetTelemetryLegacy(w http.ResponseWriter, r *http.Request) {
	traces, _, err := h.traceService.ListTraces(domain.TraceFilter{Limit: 500})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	spans := make([]*domain.Span, 0, len(traces)*4)
	for _, t := range traces {
		spans = append(spans, t.Spans...)
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"spans": spans,
		"logs":  []any{},
	})
}

