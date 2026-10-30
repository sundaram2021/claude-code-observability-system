package service_test

import (
	"testing"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
	"github.com/claude-code/observability/internal/service"
)

func TestScoreService_RecordAndLinkToTrace(t *testing.T) {
	traceStore := repository.NewMemoryStore(10, 10)
	scoreStore := repository.NewScoreStore(10)
	scoreSvc := service.NewScoreService(scoreStore, traceStore)

	// Save trace
	trace := &domain.Trace{ID: "trc_eval_1", Name: "Code generation turn"}
	_ = traceStore.SaveTrace(trace)

	// Record score with empty ID (should auto-generate ID)
	score := &domain.Score{
		TraceID: "trc_eval_1",
		Name:    "relevance",
		Value:   0.9,
		Comment: "Spot on code snippet",
	}

	if err := scoreSvc.RecordScore(score); err != nil {
		t.Fatalf("unexpected error recording score: %v", err)
	}

	if score.ID == "" {
		t.Errorf("expected auto-generated ID for score")
	}

	// Verify linked to trace
	fetchedTrace, _ := traceStore.GetTrace("trc_eval_1")
	if len(fetchedTrace.Scores) != 1 || fetchedTrace.Scores[0].Name != "relevance" {
		t.Errorf("expected score attached to trace")
	}

	// Retrieve trace scores
	scores, err := scoreSvc.GetTraceScores("trc_eval_1")
	if err != nil || len(scores) != 1 {
		t.Errorf("unexpected trace scores: %v", scores)
	}

	// List all scores
	all, err := scoreSvc.ListScores()
	if err != nil || len(all) != 1 {
		t.Errorf("unexpected all scores: %v", all)
	}

	// Invalid score validation
	if err := scoreSvc.RecordScore(nil); err != service.ErrInvalidScore {
		t.Errorf("expected ErrInvalidScore for nil score")
	}
	if err := scoreSvc.RecordScore(&domain.Score{TraceID: ""}); err != service.ErrInvalidScore {
		t.Errorf("expected ErrInvalidScore for empty traceId")
	}
}
