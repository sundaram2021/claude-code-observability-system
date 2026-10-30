package handler

import (
	"net/http"

	"github.com/claude-code/observability/internal/service"
)

// MetricsHandler provides aggregated telemetry and financial performance statistics.
type MetricsHandler struct {
	metricsService *service.MetricsService
}

// NewMetricsHandler creates an instance of MetricsHandler.
func NewMetricsHandler(metricsService *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
	}
}

// GetMetrics handles GET /api/v1/metrics.
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, _ *http.Request) {
	metrics, err := h.metricsService.GetMetrics()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, metrics, 1)
}

