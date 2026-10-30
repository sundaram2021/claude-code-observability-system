package repository

import (
	"sync"

	"github.com/claude-code/observability/internal/domain"
)

// ScoreStore provides thread-safe in-memory storage for evaluation scores.
type ScoreStore struct {
	mu           sync.RWMutex
	scores       map[string]*domain.Score
	traceIndices map[string][]string
	scoreOrder   []string
	maxScores    int
}

// NewScoreStore initializes an in-memory score repository with capacity bounds.
func NewScoreStore(maxScores int) *ScoreStore {
	if maxScores <= 0 {
		maxScores = 5000
	}
	return &ScoreStore{
		scores:       make(map[string]*domain.Score, maxScores),
		traceIndices: make(map[string][]string, maxScores),
		scoreOrder:   make([]string, 0, maxScores),
		maxScores:    maxScores,
	}
}

// SaveScore stores a new evaluation score and indexes it by trace.
func (s *ScoreStore) SaveScore(score *domain.Score) error {
	if score == nil || score.ID == "" || score.TraceID == "" {
		return ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.scores[score.ID]; !exists {
		if len(s.scoreOrder) >= s.maxScores {
			oldest := s.scoreOrder[0]
			s.scoreOrder = s.scoreOrder[1:]
			delete(s.scores, oldest)
		}
		s.scoreOrder = append(s.scoreOrder, score.ID)
	}

	s.scores[score.ID] = score
	s.traceIndices[score.TraceID] = append(s.traceIndices[score.TraceID], score.ID)

	return nil
}

// GetScores retrieves all scores recorded for a given trace.
func (s *ScoreStore) GetScores(traceID string) ([]*domain.Score, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, exists := s.traceIndices[traceID]
	if !exists || len(ids) == 0 {
		return []*domain.Score{}, nil
	}

	result := make([]*domain.Score, 0, len(ids))
	for _, id := range ids {
		if score, ok := s.scores[id]; ok {
			result = append(result, score)
		}
	}
	return result, nil
}

// ListAllScores returns all recorded evaluation scores.
func (s *ScoreStore) ListAllScores() ([]*domain.Score, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.Score, 0, len(s.scores))
	for _, id := range s.scoreOrder {
		if score, exists := s.scores[id]; exists {
			result = append(result, score)
		}
	}
	return result, nil
}
