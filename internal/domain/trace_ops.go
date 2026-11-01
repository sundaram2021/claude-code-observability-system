package domain

import (
	"strings"
	"time"
)

// IsRoot returns true if the span has no parent.
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

// EnrichPromptAndOutput extracts user prompt and assistant output from child spans.
func (t *Trace) EnrichPromptAndOutput() {
	if t == nil {
		return
	}
	if t.UserPrompt == "" {
		for _, sp := range t.Spans {
			if sp == nil || sp.Metadata == nil {
				continue
			}
			for _, k := range []string{"user_prompt", "prompt", "gen_ai.prompt", "query"} {
				if val := sp.Metadata[k]; val != "" {
					t.UserPrompt = val
					break
				}
			}
			if t.UserPrompt != "" {
				break
			}
		}
	}
	if t.UserOutput == "" {
		for _, sp := range t.Spans {
			if sp == nil || sp.Metadata == nil {
				continue
			}
			for _, k := range []string{"response", "assistant_response", "text", "content"} {
				if val := sp.Metadata[k]; val != "" && !strings.HasPrefix(val, "claude_code.") {
					t.UserOutput = val
					break
				}
			}
			if t.UserOutput != "" {
				break
			}
		}
	}
	if t.UserPrompt != "" {
		if t.Name == "" || t.Name == "Claude Code Turn" || strings.HasPrefix(t.Name, "claude_code.") || strings.HasPrefix(t.Name, "Claude Code (") {
			t.Name = t.UserPrompt
		}
	}
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

// IsCorrelatedWith checks if trace t and cand belong to the same execution turn.
func (t *Trace) IsCorrelatedWith(cand *Trace) bool {
	if t == nil || cand == nil || t.SessionID == "" || t.SessionID != cand.SessionID {
		return false
	}
	if t.UserPrompt != "" && cand.UserPrompt != "" && t.UserPrompt != cand.UserPrompt {
		return false
	}
	spanIDs := make(map[string]bool, len(t.Spans)*2)
	for _, s := range t.Spans {
		if s.ID != "" { spanIDs[s.ID] = true }
		if s.ParentID != "" { spanIDs[s.ParentID] = true }
	}
	for _, s := range cand.Spans {
		if spanIDs[s.ID] || (s.ParentID != "" && spanIDs[s.ParentID]) {
			return true
		}
	}
	if t.UserPrompt != "" && cand.UserPrompt != "" && t.UserPrompt == cand.UserPrompt {
		return true
	}
	if t.StartTime.IsZero() || cand.StartTime.IsZero() {
		return true
	}
	diff := cand.StartTime.Sub(t.StartTime)
	if diff < 0 { diff = -diff }
	if diff < 5*time.Minute {
		return true
	}
	if !t.EndTime.IsZero() && !cand.EndTime.IsZero() {
		if !(t.EndTime.Before(cand.StartTime) || cand.EndTime.Before(t.StartTime)) {
			return true
		}
	}
	return false
}
