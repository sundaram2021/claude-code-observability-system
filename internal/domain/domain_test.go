package domain_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
)

func TestSpan_IsRootAndCalculateDuration(t *testing.T) {
	root := &domain.Span{ID: "sp_1", ParentID: ""}
	child := &domain.Span{ID: "sp_2", ParentID: "sp_1"}

	if !root.IsRoot() {
		t.Errorf("expected root.IsRoot() to be true")
	}
	if child.IsRoot() {
		t.Errorf("expected child.IsRoot() to be false")
	}

	now := time.Now()
	span := &domain.Span{
		StartTime: now,
		EndTime:   now.Add(150 * time.Millisecond),
	}
	span.CalculateDuration()
	if span.DurationMs < 149.0 || span.DurationMs > 152.0 {
		t.Errorf("expected ~150ms duration, got %f", span.DurationMs)
	}
}

func TestSession_UpdateTotalsAndSummary(t *testing.T) {
	sess := &domain.Session{
		ID:        "sess_01",
		Project:   "test-proj",
		GitBranch: "main",
		Status:    domain.StatusSuccess,
	}

	now := time.Now()
	trace1 := &domain.Trace{
		TotalCost: 0.05,
		EndTime:   now,
		Status:    domain.StatusSuccess,
		TotalTokens: domain.TokenUsage{
			InputTokens:          1000,
			OutputTokens:         200,
			CacheReadInputTokens: 4000,
			TotalTokens:          5200,
		},
	}
	sess.UpdateTotals(trace1)

	if sess.TotalTurns != 1 {
		t.Errorf("expected 1 turn, got %d", sess.TotalTurns)
	}
	if sess.TotalCost != 0.05 {
		t.Errorf("expected 0.05 cost, got %f", sess.TotalCost)
	}
	// cache hit rate: 4000 / (1000 + 4000) = 80%
	if sess.CacheHitRate != 80.0 {
		t.Errorf("expected 80%% cache hit rate, got %f", sess.CacheHitRate)
	}

	traceErr := &domain.Trace{
		Status: domain.StatusError,
	}
	sess.UpdateTotals(traceErr)
	if sess.Status != domain.StatusError {
		t.Errorf("expected session status error after error trace")
	}

	summary := sess.ToSummary()
	if summary.ID != "sess_01" || summary.TotalTurns != 2 {
		t.Errorf("unexpected summary data: %+v", summary)
	}
}

func TestSystemMetrics_AddSpanAndFinalize(t *testing.T) {
	m := &domain.SystemMetrics{TotalTraces: 2}

	s1 := &domain.Span{
		CostUSD: 0.02,
		Status:  domain.StatusSuccess,
		Tokens: domain.TokenUsage{
			InputTokens:          500,
			OutputTokens:         100,
			CacheReadInputTokens: 1500,
			TotalTokens:          2100,
		},
	}
	s2 := &domain.Span{
		CostUSD: 0.01,
		Status:  domain.StatusError,
		Tokens:  domain.TokenUsage{InputTokens: 300, OutputTokens: 50, TotalTokens: 350},
	}

	m.AddSpan(s1)
	m.AddSpan(s2)
	m.FinalizeRates()

	if m.TotalSpans != 2 {
		t.Errorf("expected 2 spans, got %d", m.TotalSpans)
	}
	if m.ErrorCount != 1 {
		t.Errorf("expected 1 error count, got %d", m.ErrorCount)
	}
	if m.ErrorRate != 50.0 {
		t.Errorf("expected 50%% error rate, got %f", m.ErrorRate)
	}
	// cache hit: 1500 / (800 + 1500) = 65.217%
	if m.CacheHitRate < 65.0 || m.CacheHitRate > 66.0 {
		t.Errorf("expected ~65.2%% cache hit rate, got %f", m.CacheHitRate)
	}
}

func TestTraceFilter_ParseAndMatches(t *testing.T) {
	q := url.Values{}
	q.Set("sessionId", "s_123")
	q.Set("status", "error")
	q.Set("search", "deploy")
	q.Set("limit", "25")
	q.Set("offset", "10")

	f := domain.ParseTraceFilter(q)
	if f.SessionID != "s_123" || f.Status != domain.StatusError || f.Limit != 25 || f.Offset != 10 {
		t.Errorf("unexpected parsed filter: %+v", f)
	}

	tMatch := &domain.Trace{
		SessionID: "s_123",
		Status:    domain.StatusError,
		Name:      "Deploy to prod",
	}
	if !f.Matches(tMatch) {
		t.Errorf("expected filter to match trace")
	}

	tMismatchSess := &domain.Trace{SessionID: "s_other", Status: domain.StatusError, Name: "Deploy"}
	if f.Matches(tMismatchSess) {
		t.Errorf("expected session mismatch")
	}

	tMismatchStatus := &domain.Trace{SessionID: "s_123", Status: domain.StatusSuccess, Name: "Deploy"}
	if f.Matches(tMismatchStatus) {
		t.Errorf("expected status mismatch")
	}

	tMismatchSearch := &domain.Trace{SessionID: "s_123", Status: domain.StatusError, Name: "Build only"}
	if f.Matches(tMismatchSearch) {
		t.Errorf("expected search mismatch")
	}
}

