package ingestion

import "time"

// ClaudeHookType defines the lifecycle event type from Claude Code hooks.
type ClaudeHookType string

const (
	HookSessionStart ClaudeHookType = "session_start"
	HookSessionEnd   ClaudeHookType = "session_end"
	HookPreRun       ClaudeHookType = "prerun"
	HookPostRun      ClaudeHookType = "postrun"
	HookPreTool      ClaudeHookType = "pretool"
	HookPostTool     ClaudeHookType = "posttool"
)

// ClaudeEventPayload represents an incoming event from Claude Code hook scripts.
type ClaudeEventPayload struct {
	SessionID string            `json:"sessionId"`
	Project   string            `json:"project"`
	GitBranch string            `json:"gitBranch"`
	HookType  ClaudeHookType    `json:"hookType"`
	TurnID    string            `json:"turnId"`
	Prompt    string            `json:"prompt,omitempty"`
	Response  string            `json:"response,omitempty"`
	Model     string            `json:"model,omitempty"`
	ToolName  string            `json:"toolName,omitempty"`
	ToolInput string            `json:"toolInput,omitempty"`
	ToolOut   string            `json:"toolOutput,omitempty"`
	ExitCode  int               `json:"exitCode,omitempty"`
	IsError   bool              `json:"isError,omitempty"`
	Duration  float64           `json:"durationMs,omitempty"`
	Tokens    ClaudeTokenUsage  `json:"tokens,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ClaudeTokenUsage matches token breakdowns from Anthropic API.
type ClaudeTokenUsage struct {
	InputTokens        int `json:"inputTokens"`
	OutputTokens       int `json:"outputTokens"`
	CacheReadTokens    int `json:"cacheReadTokens"`
	CacheCreateTokens  int `json:"cacheCreateTokens"`
}

