package repository

import (
	"time"

	"github.com/claude-code/observability/internal/domain"
)

// ListTraces filters and returns traces according to query parameters.
func (s *MemoryStore) ListTraces(filter domain.TraceFilter) ([]*domain.Trace, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	allTraces := s.traces.All()
	filtered := make([]*domain.Trace, 0, len(allTraces))

	for _, t := range allTraces {
		if t != nil && s.traceIndex[t.ID] != nil && filter.Matches(t) {
			filtered = append(filtered, t)
		}
	}

	total := len(filtered)
	start := filter.Offset
	if start >= total {
		return []*domain.Trace{}, total, nil
	}

	end := total
	if filter.Limit > 0 && start+filter.Limit < total {
		end = start + filter.Limit
	}

	return filtered[start:end], total, nil
}

// SaveSession registers or updates a Claude Code session.
func (s *MemoryStore) SaveSession(sess *domain.Session) error {
	if sess == nil || sess.ID == "" {
		return ErrInvalid
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sess.ID]; !exists {
		if len(s.sessionOrder) >= s.maxSessions {
			oldest := s.sessionOrder[0]
			s.sessionOrder = s.sessionOrder[1:]
			delete(s.sessions, oldest)
		}
		s.sessionOrder = append(s.sessionOrder, sess.ID)
	}

	s.sessions[sess.ID] = sess
	return nil
}

// GetSession retrieves a Claude Code session by ID.
func (s *MemoryStore) GetSession(id string) (*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[id]
	if !exists {
		return nil, ErrNotFound
	}
	return sess, nil
}

// ListSessions returns recent sessions as summaries with pagination.
func (s *MemoryStore) ListSessions(limit, offset int) ([]domain.SessionSummary, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.sessionOrder)
	if offset >= total {
		return []domain.SessionSummary{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	summaries := make([]domain.SessionSummary, 0, end-offset)
	// Iterate backwards from newest to oldest
	for i := total - 1 - offset; i >= total-end; i-- {
		id := s.sessionOrder[i]
		if sess, exists := s.sessions[id]; exists {
			summaries = append(summaries, sess.ToSummary())
		}
	}

	return summaries, total, nil
}

// updateSessionUnderLock updates or creates a session from an incoming trace.
func (s *MemoryStore) updateSessionUnderLock(trace *domain.Trace) {
	sess, exists := s.sessions[trace.SessionID]
	if !exists {
		sess = &domain.Session{
			ID:        trace.SessionID,
			Project:   trace.Project,
			CreatedAt: trace.StartTime,
			UpdatedAt: trace.EndTime,
			Status:    domain.StatusSuccess,
		}
		if sess.CreatedAt.IsZero() {
			sess.CreatedAt = time.Now()
		}
		if len(s.sessionOrder) >= s.maxSessions {
			oldest := s.sessionOrder[0]
			s.sessionOrder = s.sessionOrder[1:]
			delete(s.sessions, oldest)
		}
		s.sessionOrder = append(s.sessionOrder, sess.ID)
		s.sessions[sess.ID] = sess
	}

	sess.UpdateTotals(trace)
}

