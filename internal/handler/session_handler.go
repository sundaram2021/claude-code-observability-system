package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

// SessionHandler handles Claude Code session HTTP requests.
type SessionHandler struct {
	sessService *service.SessionService
}

// NewSessionHandler creates an instance of SessionHandler.
func NewSessionHandler(sessService *service.SessionService) *SessionHandler {
	return &SessionHandler{
		sessService: sessService,
	}
}

// ListSessions handles GET /api/v1/sessions.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	sessions, total, err := h.sessService.ListSessions(limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusOK, sessions, total)
}

// GetSession handles GET /api/v1/sessions/{id}.
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "missing session id")
		return
	}

	session, traces, err := h.sessService.GetSessionDetail(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "session not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusOK, map[string]any{
		"session": session,
		"traces":  traces,
	}, 1)
}

