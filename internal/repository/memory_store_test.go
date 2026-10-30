package repository_test

import (
	"testing"
	"time"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

func TestMemoryStore_SaveAndGetTrace(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)

	trace := &domain.Trace{
		ID:        "trc_123",
		SessionID: "sess_abc",
		Name:      "test-trace",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Spans: []*domain.Span{
			{ID: "sp_1", TraceID: "trc_123", Name: "root-span"},
		},
	}

	if err := store.SaveTrace(trace); err != nil {
		t.Fatalf("unexpected error saving trace: %v", err)
	}

	fetched, err := store.GetTrace("trc_123")
	if err != nil {
		t.Fatalf("unexpected error getting trace: %v", err)
	}
	if fetched.ID != "trc_123" || fetched.Name != "test-trace" {
		t.Errorf("mismatched trace data: %+v", fetched)
	}

	// Verify span index
	span, err := store.GetSpan("sp_1")
	if err != nil || span.Name != "root-span" {
		t.Errorf("failed to retrieve indexed span: %v", err)
	}
}

func TestMemoryStore_EvictionUnderCapacity(t *testing.T) {
	store := repository.NewMemoryStore(2, 2)

	t1 := &domain.Trace{ID: "t1", Name: "Trace 1"}
	t2 := &domain.Trace{ID: "t2", Name: "Trace 2"}
	t3 := &domain.Trace{ID: "t3", Name: "Trace 3"}

	_ = store.SaveTrace(t1)
	_ = store.SaveTrace(t2)
	_ = store.SaveTrace(t3) // evicts t1

	if _, err := store.GetTrace("t1"); err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound for evicted trace t1, got %v", err)
	}

	if _, err := store.GetTrace("t2"); err != nil {
		t.Errorf("expected t2 to exist, got %v", err)
	}
	if _, err := store.GetTrace("t3"); err != nil {
		t.Errorf("expected t3 to exist, got %v", err)
	}
}

func TestMemoryStore_ListAndFilterTraces(t *testing.T) {
	store := repository.NewMemoryStore(10, 10)

	_ = store.SaveTrace(&domain.Trace{ID: "t1", SessionID: "s1", Status: domain.StatusSuccess, Name: "Run tests"})
	_ = store.SaveTrace(&domain.Trace{ID: "t2", SessionID: "s1", Status: domain.StatusError, Name: "Edit file"})
	_ = store.SaveTrace(&domain.Trace{ID: "t3", SessionID: "s2", Status: domain.StatusSuccess, Name: "Bash build"})

	// Filter by SessionID
	traces, total, _ := store.ListTraces(domain.TraceFilter{SessionID: "s1", Limit: 10})
	if total != 2 || len(traces) != 2 {
		t.Errorf("expected 2 traces for session s1, got %d", total)
	}

	// Filter by Search keyword
	searched, _, _ := store.ListTraces(domain.TraceFilter{Search: "build", Limit: 10})
	if len(searched) != 1 || searched[0].ID != "t3" {
		t.Errorf("expected trace t3 for search 'build'")
	}
}

