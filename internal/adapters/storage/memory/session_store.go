package memory

import (
	"errors"
	"sync"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[domain.SessionID]*domain.Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[domain.SessionID]*domain.Session),
	}
}

func (s *SessionStore) CreateSession(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; exists {
		return errors.New("session already exists")
	}

	s.sessions[session.ID] = session
	return nil
}

func (s *SessionStore) UpdateSession(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; !exists {
		return errors.New("session not found")
	}

	s.sessions[session.ID] = session
	return nil
}

func (s *SessionStore) GetSession(id domain.SessionID) (*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}

	return sess, nil
}

func (S *SessionStore) ListSessionsByUser(userID domain.UserID, limit int) ([]*domain.Session, error) {
	S.mu.RLock()
	defer S.mu.RUnlock()

	var result []*domain.Session
	for _, sess := range S.sessions {
		if sess.UserID == userID {
			result = append(result, sess)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}
