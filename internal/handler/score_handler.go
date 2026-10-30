package handler

import (
	"net/http"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/service"
)

// ScoreHandler provides REST endpoints for recording and querying evaluation scores.
type ScoreHandler struct {
	scoreService *service.ScoreService
}

// NewScoreHandler creates an instance of ScoreHandler.
func NewScoreHandler(scoreService *service.ScoreService) *ScoreHandler {
	return &ScoreHandler{scoreService: scoreService}
}

// CreateScore handles POST /api/v1/scores.
func (h *ScoreHandler) CreateScore(w http.ResponseWriter, r *http.Request) {
	var score domain.Score
	if err := ReadJSON(r, &score); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed score JSON: "+err.Error())
		return
	}

	if err := h.scoreService.RecordScore(&score); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	WriteSuccess(w, http.StatusCreated, score, 1)
}

// ListScores handles GET /api/v1/scores.
func (h *ScoreHandler) ListScores(w http.ResponseWriter, r *http.Request) {
	traceID := r.URL.Query().Get("traceId")
	if traceID != "" {
		scores, err := h.scoreService.GetTraceScores(traceID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteSuccess(w, http.StatusOK, scores, len(scores))
		return
	}

	scores, err := h.scoreService.ListScores()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteSuccess(w, http.StatusOK, scores, len(scores))
}

// GetTraceScores handles GET /api/v1/traces/{id}/scores.
func (h *ScoreHandler) GetTraceScores(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "missing trace id")
		return
	}

	scores, err := h.scoreService.GetTraceScores(id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusOK, scores, len(scores))
}

