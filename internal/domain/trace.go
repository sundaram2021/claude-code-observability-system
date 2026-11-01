package domain

import (
	"time"
)

// SpanKind represents the role of a span in a Claude Code execution.
type SpanKind string

const (
	SpanKindWorkflow   SpanKind = "WORKFLOW"   // Top-level agent session/turn
	SpanKindGeneration SpanKind = "GENERATION" // LLM model generation
	SpanKindTool       SpanKind = "TOOL"       // Claude tool execution (bash, edit, etc.)
	SpanKindEvent      SpanKind = "EVENT"      // Point-in-time lifecycle event
)

// SpanStatus represents the completion state of a span.
type SpanStatus string

const (
	StatusSuccess SpanStatus = "SUCCESS"
	StatusError   SpanStatus = "ERROR"
	StatusRunning SpanStatus = "RUNNING"
)

// TokenUsage tracks token consumption and prompt caching efficiency.
type TokenUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CacheCreation5mTokens    int     `json:"cacheCreation5mTokens,omitempty"`
	CacheCreation1hTokens    int     `json:"cacheCreation1hTokens,omitempty"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	TotalTokens              int     `json:"totalTokens"`
	ContextTokens            int     `json:"contextTokens"`
	ContextWindowLimit       int     `json:"contextWindowLimit"`
	ContextSaturationPct     float64 `json:"contextSaturationPct"`
}

// ToolCall contains specific execution details for Claude Code tool invocations.
type ToolCall struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode,omitempty"`
	IsError  bool   `json:"isError"`
}

// Span represents a single unit of work in Claude Code observability.
type Span struct {
	ID           string            `json:"id"`
	TraceID      string            `json:"traceId"`
	ParentID     string            `json:"parentId,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
	Name         string            `json:"name"`
	Kind         SpanKind          `json:"kind"`
	Status       SpanStatus        `json:"status"`
	Model        string            `json:"model,omitempty"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime"`
	DurationMs   float64           `json:"durationMs"`
	TTFTMs       float64           `json:"ttftMs,omitempty"`
	TokensPerSec float64           `json:"tokensPerSec,omitempty"`
	Effort       string            `json:"effort,omitempty"`
	Speed        string            `json:"speed,omitempty"`
	Tokens       TokenUsage        `json:"tokens"`
	CostUSD      float64           `json:"costUsd"`
	Tool         *ToolCall         `json:"tool,omitempty"`
	ErrorMsg     string            `json:"errorMsg,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Children     []*Span           `json:"children,omitempty"`
}

// Trace represents an entire distributed trace or Claude Code turn.
type Trace struct {
	ID                   string            `json:"id"`
	SessionID            string            `json:"sessionId,omitempty"`
	Project              string            `json:"project,omitempty"`
	Name                 string            `json:"name"`
	UserPrompt           string            `json:"userPrompt,omitempty"`
	UserOutput           string            `json:"userOutput,omitempty"`
	Status               SpanStatus        `json:"status"`
	UserID               string            `json:"userId,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Scores               []*Score          `json:"scores,omitempty"`
	StartTime            time.Time         `json:"startTime"`
	EndTime              time.Time         `json:"endTime"`
	DurationMs           float64           `json:"durationMs"`
	AvgTTFTMs            float64           `json:"avgTtftMs,omitempty"`
	TokensPerSec         float64           `json:"tokensPerSec,omitempty"`
	CacheSavingsUSD      float64           `json:"cacheSavingsUsd"`
	CacheMultiplier      float64           `json:"cacheMultiplier"`
	ContextSaturationPct float64           `json:"contextSaturationPct"`
	ToolCount            int               `json:"toolCount"`
	ToolDurationMs       float64           `json:"toolDurationMs"`
	GenerationDurationMs float64           `json:"generationDurationMs"`
	SelfCorrected        bool              `json:"selfCorrected"`
	RootSpan             *Span             `json:"rootSpan,omitempty"`
	Spans                []*Span           `json:"spans"`
	TotalTokens          TokenUsage        `json:"totalTokens"`
	TotalCost            float64           `json:"totalCost"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}
