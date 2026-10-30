package domain

import (
	"time"
)

// Session represents a Claude Code interactive agent session.
type Session struct {
	ID           string            `json:"id"`
	Project      string            `json:"project"`
	GitBranch    string            `json:"gitBranch,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	TotalTurns   int               `json:"totalTurns"`
	TotalCost    float64           `json:"totalCost"`
	Tokens       TokenUsage        `json:"tokens"`
	CacheHitRate float64           `json:"cacheHitRate"`
	Status       SpanStatus        `json:"status"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SessionSummary provides a compact view for list displays.
type SessionSummary struct {
	ID           string     `json:"id"`
	Project      string     `json:"project"`
	GitBranch    string     `json:"gitBranch"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	TotalTurns   int        `json:"totalTurns"`
	TotalCost    float64    `json:"totalCost"`
	TotalTokens  int        `json:"totalTokens"`
	CacheHitRate float64    `json:"cacheHitRate"`
	Status       SpanStatus `json:"status"`
}

// UpdateTotals recalculates session totals from a new trace.
func (s *Session) UpdateTotals(t *Trace) {
	s.TotalTurns++
	s.TotalCost += t.TotalCost
	s.Tokens.InputTokens += t.TotalTokens.InputTokens
	s.Tokens.OutputTokens += t.TotalTokens.OutputTokens
	s.Tokens.CacheReadInputTokens += t.TotalTokens.CacheReadInputTokens
	s.Tokens.CacheCreationInputTokens += t.TotalTokens.CacheCreationInputTokens
	s.Tokens.TotalTokens += t.TotalTokens.TotalTokens

	totalPrompt := s.Tokens.InputTokens + s.Tokens.CacheReadInputTokens
	if totalPrompt > 0 {
		s.CacheHitRate = (float64(s.Tokens.CacheReadInputTokens) / float64(totalPrompt)) * 100.0
	}
	s.UpdatedAt = t.EndTime
	if t.Status == StatusError {
		s.Status = StatusError
	}
}

// ToSummary converts a session into its lightweight summary format.
func (s *Session) ToSummary() SessionSummary {
	return SessionSummary{
		ID:           s.ID,
		Project:      s.Project,
		GitBranch:    s.GitBranch,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
		TotalTurns:   s.TotalTurns,
		TotalCost:    s.TotalCost,
		TotalTokens:  s.Tokens.TotalTokens,
		CacheHitRate: s.CacheHitRate,
		Status:       s.Status,
	}
}

