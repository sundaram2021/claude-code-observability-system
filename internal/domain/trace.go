package domain

import (
	"strings"
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

// IsRoot returns true if the span is a root span.
func (s *Span) IsRoot() bool {
	return s.ParentID == ""
}

// CalculateDuration computes duration in milliseconds between end and start.
func (s *Span) CalculateDuration() {
	if !s.EndTime.IsZero() && !s.StartTime.IsZero() {
		s.DurationMs = float64(s.EndTime.Sub(s.StartTime).Microseconds()) / 1000.0
	}
}

// AddScore attaches an evaluation score to the trace.
func (t *Trace) AddScore(score *Score) {
	if score != nil {
		t.Scores = append(t.Scores, score)
	}
}

// IsStartupPlumbing returns true if the trace only contains zero-duration lifecycle hook events.
func (t *Trace) IsStartupPlumbing() bool {
	if t == nil || t.UserPrompt != "" || t.UserOutput != "" || len(t.Spans) == 0 {
		return false
	}
	for _, sp := range t.Spans {
		if sp.Kind == SpanKindGeneration || sp.Kind == SpanKindTool || sp.Model != "" || sp.Tool != nil || sp.DurationMs > 50 {
			return false
		}
		name := sp.Name
		if !strings.HasPrefix(name, "Event: hook_") && !strings.HasPrefix(name, "Event: managed_settings_") && !strings.HasPrefix(name, "Event: plugin_loaded") {
			return false
		}
	}
	return true
}

// MergeFrom merges spans, metadata, prompt, and output from another trace into t.
func (t *Trace) MergeFrom(src *Trace) {
	if src == nil {
		return
	}
	spanMap := make(map[string]*Span, len(t.Spans))
	for _, s := range t.Spans {
		spanMap[s.ID] = s
	}
	for _, s := range src.Spans {
		if ex, ok := spanMap[s.ID]; ok {
			if s.DurationMs > ex.DurationMs {
				ex.DurationMs, ex.StartTime, ex.EndTime = s.DurationMs, s.StartTime, s.EndTime
			}
			if ex.Kind == SpanKindWorkflow && s.Kind != SpanKindWorkflow { ex.Kind = s.Kind }
			if ex.Model == "" && s.Model != "" { ex.Model = s.Model }
			if ex.Tool == nil && s.Tool != nil { ex.Tool = s.Tool }
			if s.Tokens.TotalTokens > ex.Tokens.TotalTokens { ex.Tokens = s.Tokens }
			if ex.ParentID == "" && s.ParentID != "" { ex.ParentID = s.ParentID }
			if s.Status == StatusError { ex.Status = StatusError }
		} else {
			t.Spans = append(t.Spans, s)
			spanMap[s.ID] = s
		}
	}
	if !src.StartTime.IsZero() && (t.StartTime.IsZero() || src.StartTime.Before(t.StartTime)) {
		t.StartTime = src.StartTime
	}
	if src.EndTime.After(t.EndTime) {
		t.EndTime = src.EndTime
	}
	if t.UserPrompt == "" && src.UserPrompt != "" {
		t.UserPrompt = src.UserPrompt
	}
	if t.UserOutput == "" && src.UserOutput != "" {
		t.UserOutput = src.UserOutput
	}
	if (t.Name == "" || t.Name == "Claude Code Turn" || strings.HasPrefix(t.Name, "claude_code.")) && src.Name != "" && !strings.HasPrefix(src.Name, "claude_code.") {
		t.Name = src.Name
	}
	if t.UserPrompt != "" {
		t.Name = t.UserPrompt
	}
	if src.Status == StatusError {
		t.Status = StatusError
	}
}
