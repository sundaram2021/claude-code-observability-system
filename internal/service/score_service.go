package service

import (
	"errors"

	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

var ErrInvalidScore = errors.New("score name, traceId, and id are required")

// ScoreService coordinates evaluation scores and trace attachments.
type ScoreService struct {
	scoreRepo repository.ScoreRepository
	traceRepo repository.TraceRepository
}

// NewScoreService creates an instance of ScoreService.
func NewScoreService(scoreRepo repository.ScoreRepository, traceRepo repository.TraceRepository) *ScoreService {
	return &ScoreService{
		scoreRepo: scoreRepo,
		traceRepo: traceRepo,
	}
}

// RecordScore validates, saves, and links an evaluation score to its trace.
func (s *ScoreService) RecordScore(score *domain.Score) error {
	if score == nil || score.TraceID == "" || score.Name == "" {
		return ErrInvalidScore
	}
	if score.ID == "" {
		score.ID = generateID("score_")
	}

	if err := s.scoreRepo.SaveScore(score); err != nil {
		return err
	}

	// Attach score to trace if present
	if trace, err := s.traceRepo.GetTrace(score.TraceID); err == nil && trace != nil {
		trace.AddScore(score)
	}

	return nil
}

// GetTraceScores retrieves scores for a specific trace.
func (s *ScoreService) GetTraceScores(traceID string) ([]*domain.Score, error) {
	return s.scoreRepo.GetScores(traceID)
}

// ListScores returns all evaluation scores.
func (s *ScoreService) ListScores() ([]*domain.Score, error) {
	return s.scoreRepo.ListAllScores()
}
