package domain

import "time"

// Score represents a qualitative evaluation metric attached to a trace or span.
type Score struct {
	ID        string    `json:"id"`
	TraceID   string    `json:"traceId"`
	SpanID    string    `json:"spanId,omitempty"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Comment   string    `json:"comment,omitempty"`
	Source    string    `json:"source,omitempty"` // "USER", "EVAL", "AGENT"
	CreatedAt time.Time `json:"createdAt"`
}

// NewScore initializes an evaluation score with a timestamp.
func NewScore(id, traceID, spanID, name string, value float64, comment, source string) *Score {
	if source == "" {
		source = "USER"
	}
	return &Score{
		ID:        id,
		TraceID:   traceID,
		SpanID:    spanID,
		Name:      name,
		Value:     value,
		Comment:   comment,
		Source:    source,
		CreatedAt: time.Now(),
	}
}
