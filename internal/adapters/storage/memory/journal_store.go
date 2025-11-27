package memory

import (
	"sync"
	"time"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

// MemoryJournalStore is a simple in-memory implementation of domain.JournalStore.
// It is NOT persistent and is only suitable for development / local mode.
type MemoryJournalStore struct {
	mu       sync.RWMutex
	entries  map[domain.JournalEntryID]*domain.JournalEntry
	byUserID map[domain.UserID][]domain.JournalEntryID
}

// NewJournalStore creates a new in-memory JournalStore.
func NewJournalStore() *MemoryJournalStore {
	return &MemoryJournalStore{
		entries:  make(map[domain.JournalEntryID]*domain.JournalEntry),
		byUserID: make(map[domain.UserID][]domain.JournalEntryID),
	}
}

// AppendJournalEntry saves a new journal entry.
func (s *MemoryJournalStore) AppendJournalEntry(entry *domain.JournalEntry) error {
	if entry == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If no ID is provided, generate a simple one based on the current time.
	if entry.ID == "" {
		entry.ID = domain.JournalEntryID(generateID(time.Now()))
	}

	s.entries[entry.ID] = entry
	s.byUserID[entry.UserID] = append(s.byUserID[entry.UserID], entry.ID)

	return nil
}

// ListJournalEntriesByUser returns the last `limit` entries for a user.
// If limit <= 0, returns all.
func (s *MemoryJournalStore) ListJournalEntriesByUser(
	userID domain.UserID,
	limit int,
) ([]*domain.JournalEntry, error) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byUserID[userID]
	if len(ids) == 0 {
		return []*domain.JournalEntry{}, nil
	}

	// If limit is not valid, use all
	if limit <= 0 || limit > len(ids) {
		limit = len(ids)
	}

	// Take the last `limit` IDs
	start := len(ids) - limit
	selected := ids[start:]

	out := make([]*domain.JournalEntry, 0, len(selected))
	for _, id := range selected {
		if e, ok := s.entries[id]; ok {
			out = append(out, e)
		}
	}

	return out, nil
}

// generateID reuses the same simple format used by conversation.Service.
func generateID(t time.Time) string {
	return t.Format("20060102150405.000000000")
}
