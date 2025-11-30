package journal

import (
	"context"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

// Service holds the logic of reading journal entries
type Service struct {
	store domain.JournalStore
}

// NewService creates a journal service from a JournalStore
func NewService(store domain.JournalStore) *Service {
	return &Service{
		store: store,
	}
}

// GetUserJournal returns the last `limit` journal entries for a user
// If limit <= 0, a reasonable default value is used.
func (s *Service) GetUserJournal(
	ctx context.Context,
	userID domain.UserID,
	limit int,
) ([]*domain.JournalEntry, error) {

	if s.store == nil {
		// In GCP mode, until we implement FirestoreJournalStore,
		// the store can be nil. We return an empty slice without error
		return []*domain.JournalEntry{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	// For now we ignore ctx because MemoryJournalStore does not use it,
	// but the interface could be extended in the future
	return s.store.ListJournalEntriesByUser(userID, limit)
}
