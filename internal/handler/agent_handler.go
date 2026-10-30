package handler

import (
	"net/http"

	"github.com/claude-code/observability/internal/agent"
)

// AgentHandler provides execution endpoints for Claude Code agent simulations and live runs.
type AgentHandler struct {
	runner *agent.Runner
}

// NewAgentHandler creates an instance of AgentHandler.
func NewAgentHandler(runner *agent.Runner) *AgentHandler {
	return &AgentHandler{
		runner: runner,
	}
}

// RunAgent handles POST /api/v1/agent/run.
func (h *AgentHandler) RunAgent(w http.ResponseWriter, r *http.Request) {
	var req agent.RunRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "malformed agent run request: "+err.Error())
		return
	}

	if req.Prompt == "" {
		WriteError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	resp, err := h.runner.Run(r.Context(), req)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteSuccess(w, http.StatusOK, resp, 1)
}

// LegacyChat handles POST /api/chat for backward compatibility.
func (h *AgentHandler) LegacyChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if err := ReadJSON(r, &body); err != nil || body.Message == "" {
		WriteError(w, http.StatusBadRequest, "invalid message")
		return
	}

	resp, err := h.runner.Run(r.Context(), agent.RunRequest{
		Prompt:   body.Message,
		Simulate: true,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"conversation": []map[string]any{
			{"role": "user", "content": body.Message},
			{"role": "assistant", "content": resp.Output},
		},
	})
}

