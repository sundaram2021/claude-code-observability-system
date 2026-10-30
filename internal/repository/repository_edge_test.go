package repository_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

func TestMemoryStore_InvalidAndNotFoundErrors(t *testing.T) {
	store := repository.NewMemoryStore(5, 5)

	// SaveTrace invalid
	if err := store.SaveTrace(nil); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for nil trace")
	}
	if err := store.SaveTrace(&domain.Trace{ID: ""}); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for empty trace ID")
	}

	// SaveSpan invalid
	if err := store.SaveSpan(nil); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for nil span")
	}
	if err := store.SaveSpan(&domain.Span{ID: ""}); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for empty span ID")
	}

	// GetSpan not found
	if _, err := store.GetSpan("non_existent"); err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent span")
	}

	// SaveSession invalid
	if err := store.SaveSession(nil); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for nil session")
	}
	if err := store.SaveSession(&domain.Session{ID: ""}); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for empty session ID")
	}

	// GetSession not found
	if _, err := store.GetSession("non_existent_sess"); err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for non-existent session")
	}
}

func TestMemoryStore_SpanUpdateAndSessionEviction(t *testing.T) {
	store := repository.NewMemoryStore(5, 2)

	// Save trace with a span
	trace := &domain.Trace{
		ID:        "t_up",
		SessionID: "s_up",
		Spans: []*domain.Span{
			{ID: "sp_1", TraceID: "t_up", Name: "original_name"},
		},
	}
	_ = store.SaveTrace(trace)

	// Update existing span
	updatedSpan := &domain.Span{ID: "sp_1", TraceID: "t_up", Name: "updated_name"}
	if err := store.SaveSpan(updatedSpan); err != nil {
		t.Fatalf("failed to update span: %v", err)
	}

	sp, err := store.GetSpan("sp_1")
	if err != nil || sp.Name != "updated_name" {
		t.Errorf("expected updated span name 'updated_name', got %s", sp.Name)
	}

	// Add new span to existing trace
	newSpan := &domain.Span{ID: "sp_new", TraceID: "t_up", Name: "added_name"}
	_ = store.SaveSpan(newSpan)
	tr, _ := store.GetTrace("t_up")
	if len(tr.Spans) != 2 {
		t.Errorf("expected 2 spans in trace, got %d", len(tr.Spans))
	}

	// Test Session eviction when maxSessions=2 is exceeded
	s1 := &domain.Session{ID: "sess_1", Project: "p1", CreatedAt: time.Now()}
	s2 := &domain.Session{ID: "sess_2", Project: "p2", CreatedAt: time.Now()}
	s3 := &domain.Session{ID: "sess_3", Project: "p3", CreatedAt: time.Now()}

	_ = store.SaveSession(s1)
	_ = store.SaveSession(s2)
	_ = store.SaveSession(s3) // evicts sess_1

	if _, err := store.GetSession("sess_1"); err != repository.ErrNotFound {
		t.Errorf("expected sess_1 to be evicted")
	}

	summaries, total, _ := store.ListSessions(10, 0)
	if total != 2 || len(summaries) != 2 {
		t.Errorf("expected 2 sessions, got %d", total)
	}

	// Offset past total
	emptySummaries, _, _ := store.ListSessions(10, 100)
	if len(emptySummaries) != 0 {
		t.Errorf("expected empty summaries for offset >= total")
	}

	// Test Clear()
	_ = store.Clear()
	if store.Count() != 0 {
		t.Errorf("expected count 0 after clear")
	}
}

