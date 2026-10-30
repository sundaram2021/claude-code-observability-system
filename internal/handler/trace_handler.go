package handler

import (
	"errors"
	"net/http"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

// TraceHandler handles trace querying and retrieval HTTP requests.
type TraceHandler struct {
	traceService *service.TraceService
	repo         repository.TraceRepository
}

// NewTraceHandler creates an instance of TraceHandler.
func NewTraceHandler(traceService *service.TraceService, repo repository.TraceRepository) *TraceHandler {
	return &TraceHandler{
		traceService: traceService,
		repo:         repo,
	}
}

// ListTraces handles GET /api/v1/traces.
func (h *TraceHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	filter := domain.ParseTraceFilter(r.URL.Query())
	traces, total, err := h.traceService.ListTraces(filter)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, traces, total)
}

// GetTrace handles GET /api/v1/traces/{id}.
func (h *TraceHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "missing trace id")
		return
	}

	trace, err := h.traceService.GetTrace(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "trace not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusOK, trace, 1)
}

// ClearTraces handles DELETE /api/v1/traces.
func (h *TraceHandler) ClearTraces(w http.ResponseWriter, _ *http.Request) {
	if err := h.repo.Clear(); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, map[string]string{"message": "traces cleared"}, 0)
}

// CreateTrace handles manual POST /api/v1/traces.
func (h *TraceHandler) CreateTrace(w http.ResponseWriter, r *http.Request) {
	var trace domain.Trace
	if err := ReadJSON(r, &trace); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed trace JSON: "+err.Error())
		return
	}

	if err := h.traceService.IngestTrace(&trace); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	WriteSuccess(w, http.StatusCreated, trace, 1)
}

