package memory

import (
	"sync"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

type MessageStore struct {
	mu       sync.RWMutex
	messages map[domain.SessionID][]*domain.Message
}

func NewMessageStore() *MessageStore {
	return &MessageStore{
		messages: make(map[domain.SessionID][]*domain.Message),
	}
}

func (s *MessageStore) AppendMessage(msg *domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)
	return nil
}

func (s *MessageStore) GetMessagesBySession(sessionID domain.SessionID, limit int) ([]*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.messages[sessionID]
	if limit > 0 && len(msgs) > limit {
		return msgs[len(msgs)-limit:], nil
	}
	return msgs, nil
}
