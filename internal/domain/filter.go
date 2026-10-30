package domain

import (
	"net/url"
	"strconv"
	"strings"
)

// TraceFilter defines criteria for querying stored traces.
type TraceFilter struct {
	SessionID string
	Status    SpanStatus
	Tool      string
	Model     string
	Search    string
	Limit     int
	Offset    int
}

// ParseTraceFilter extracts filter options from HTTP query parameters.
func ParseTraceFilter(q url.Values) TraceFilter {
	limit := 50
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(q.Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	return TraceFilter{
		SessionID: strings.TrimSpace(q.Get("sessionId")),
		Status:    SpanStatus(strings.ToUpper(strings.TrimSpace(q.Get("status")))),
		Tool:      strings.TrimSpace(q.Get("tool")),
		Model:     strings.TrimSpace(q.Get("model")),
		Search:    strings.TrimSpace(q.Get("search")),
		Limit:     limit,
		Offset:    offset,
	}
}

// Matches tests whether a trace matches the given filter.
func (f *TraceFilter) Matches(t *Trace) bool {
	if f.SessionID != "" && t.SessionID != f.SessionID {
		return false
	}
	if f.Status != "" && t.Status != f.Status {
		return false
	}
	if f.Search != "" {
		s := strings.ToLower(f.Search)
		if !strings.Contains(strings.ToLower(t.Name), s) &&
			!strings.Contains(strings.ToLower(t.ID), s) &&
			!strings.Contains(strings.ToLower(t.Project), s) {
			return false
		}
	}
	return true
}

