package service

import (
	"github.com/claude-code/observability/internal/domain"
	"github.com/claude-code/observability/internal/repository"
)

// SessionService manages Claude Code session records and timelines.
type SessionService struct {
	sessRepo  repository.SessionRepository
	traceRepo repository.TraceRepository
}

// NewSessionService creates an instance of SessionService.
func NewSessionService(sessRepo repository.SessionRepository, traceRepo repository.TraceRepository) *SessionService {
	return &SessionService{
		sessRepo:  sessRepo,
		traceRepo: traceRepo,
	}
}

// ListSessions retrieves paginated session summaries.
func (s *SessionService) ListSessions(limit, offset int) ([]domain.SessionSummary, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.sessRepo.ListSessions(limit, offset)
}

// GetSessionDetail fetches a session along with its associated traces.
func (s *SessionService) GetSessionDetail(id string) (*domain.Session, []*domain.Trace, error) {
	session, err := s.sessRepo.GetSession(id)
	if err != nil {
		return nil, nil, err
	}

	filter := domain.TraceFilter{
		SessionID: id,
		Limit:     100,
	}
	traces, _, err := s.traceRepo.ListTraces(filter)
	if err != nil {
		return session, nil, err
	}

	return session, traces, nil
}

