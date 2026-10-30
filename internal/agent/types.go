package agent

import "github.com/claude-code/observability/internal/domain"

// RunRequest represents a request to execute a Claude Code agent task.
type RunRequest struct {
	Prompt    string `json:"prompt"`
	SessionID string `json:"sessionId,omitempty"`
	Project   string `json:"project,omitempty"`
	Model     string `json:"model,omitempty"`
	Simulate  bool   `json:"simulate,omitempty"`
}

// RunResponse represents the execution outcome of an agent run.
type RunResponse struct {
	TraceID    string             `json:"traceId"`
	SessionID  string             `json:"sessionId"`
	Success    bool               `json:"success"`
	Output     string             `json:"output"`
	Turns      int                `json:"turns"`
	DurationMs float64            `json:"durationMs"`
	CostUSD    float64            `json:"costUsd"`
	Tokens     domain.TokenUsage  `json:"tokens"`
	Error      string             `json:"error,omitempty"`
}

// ToolDefinition describes an executable tool provided to Claude Code.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ToolExecResult holds output from executing an agent tool.
type ToolExecResult struct {
	ToolName string
	Input    string
	Output   string
	ExitCode int
	IsError  bool
}

