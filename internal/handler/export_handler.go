package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/service"
)

// ExportHandler handles exporting traces as downloadable JSON datasets.
type ExportHandler struct {
	traceService *service.TraceService
}

// NewExportHandler creates an instance of ExportHandler.
func NewExportHandler(traceService *service.TraceService) *ExportHandler {
	return &ExportHandler{traceService: traceService}
}

// ExportTraces handles GET /api/v1/traces/export.
func (h *ExportHandler) ExportTraces(w http.ResponseWriter, r *http.Request) {
	filter := domain.ParseTraceFilter(r.URL.Query())
	filter.Limit = 5000 // Allow full bulk export

	traces, total, err := h.traceService.ListTraces(filter)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("claude-traces-%s.json", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)

	payload := map[string]any{
		"version":    "1.0",
		"exportedAt": time.Now(),
		"total":      total,
		"traces":     traces,
	}

	_ = json.NewEncoder(w).Encode(payload)
}
