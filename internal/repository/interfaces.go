package repository

import (
	"errors"

	"github.com/claude-code/observability/internal/domain"
)

var (
	ErrNotFound = errors.New("entity not found")
	ErrInvalid  = errors.New("invalid entity data")
)

// TraceRepository defines operations for storing and querying traces and spans.
type TraceRepository interface {
	SaveTrace(trace *domain.Trace) error
	GetTrace(id string) (*domain.Trace, error)
	ListTraces(filter domain.TraceFilter) ([]*domain.Trace, int, error)
	SaveSpan(span *domain.Span) error
	GetSpan(id string) (*domain.Span, error)
	Count() int
	Clear() error
}

// SessionRepository defines operations for managing Claude Code sessions.
type SessionRepository interface {
	SaveSession(session *domain.Session) error
	GetSession(id string) (*domain.Session, error)
	ListSessions(limit, offset int) ([]domain.SessionSummary, int, error)
}

// ScoreRepository defines operations for managing evaluation scores.
type ScoreRepository interface {
	SaveScore(score *domain.Score) error
	GetScores(traceID string) ([]*domain.Score, error)
	ListAllScores() ([]*domain.Score, error)
}
