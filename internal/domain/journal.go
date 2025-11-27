package domain

import "time"

// JournalEntryID identifies a journal entry
type JournalEntryID string

// ActionStatus represents the status of an action in the plan
type ActionStatus string

const (
	ActionStatusPending ActionStatus = "pending"
	ActionStatusDone    ActionStatus = "done"
)

// JournalAction represents a concrete step within an action plan
type JournalAction struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Status      ActionStatus  `json:"status"`
	Notes       string        `json:"notes,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// JournalEntry represents the “long-term” summary of a session or set of sessions
type JournalEntry struct {
	ID        JournalEntryID `json:"id"`
	SessionID SessionID      `json:"session_id"`
	UserID    UserID         `json:"user_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// A brief summary of the problem worked on in the session
	ProblemSummary string `json:"problem_summary"`

	// Proposed action plan (2-4 steps, typically).
	ActionPlan []JournalAction `json:"action_plan"`

	// Final reflection (can be written by the Reflector agent or the user)
	Reflection string `json:"reflection"`

	// Emotional state before and after the session
	MoodBefore string `json:"mood_before"`
	MoodAfter  string `json:"mood_after"`
}

// JournalStore defines the minimum operations to persist the journal
type JournalStore interface {
	AppendJournalEntry(entry *JournalEntry) error
	ListJournalEntriesByUser(userID UserID, limit int) ([]*JournalEntry, error)
}
