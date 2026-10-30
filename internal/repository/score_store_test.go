package repository_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

func TestScoreStore_SaveAndRetrieve(t *testing.T) {
	store := repository.NewScoreStore(2)

	// Invalid checks
	if err := store.SaveScore(nil); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for nil score")
	}
	if err := store.SaveScore(&domain.Score{ID: ""}); err != repository.ErrInvalid {
		t.Errorf("expected ErrInvalid for empty ID")
	}

	sc1 := domain.NewScore("sc_1", "trc_1", "", "accuracy", 1.0, "perfect", "USER")
	sc2 := domain.NewScore("sc_2", "trc_1", "", "latency_rating", 0.8, "fast", "USER")
	sc3 := domain.NewScore("sc_3", "trc_2", "", "accuracy", 0.5, "partial", "USER")

	_ = store.SaveScore(sc1)
	_ = store.SaveScore(sc2)
	_ = store.SaveScore(sc3) // should evict sc1 (capacity is 2)

	scoresT1, err := store.GetScores("trc_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sc1 was evicted, sc2 remains
	if len(scoresT1) != 1 || scoresT1[0].ID != "sc_2" {
		t.Errorf("expected only sc_2 for trc_1 after eviction: %+v", scoresT1)
	}

	// Empty trace ID
	emptyScores, _ := store.GetScores("non_existent_trace")
	if len(emptyScores) != 0 {
		t.Errorf("expected empty slice for non-existent trace")
	}

	all, _ := store.ListAllScores()
	if len(all) != 2 {
		t.Errorf("expected 2 active scores, got %d", len(all))
	}
}
