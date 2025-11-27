package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

// JournalTool uses a domain.JournalStore to save reflections
// and long-term action plans.
type JournalTool struct {
	store domain.JournalStore
	now   func() time.Time
}

// NewJournalTool creates a new JournalTool.
// store can be an in-memory or Firestore implementation.
func NewJournalTool(store domain.JournalStore) *JournalTool {
	return &JournalTool{
		store: store,
		now:   time.Now,
	}
}

func (t *JournalTool) Name() string {
	return "journal_store"
}

// Call expects an input with this shape:
//
// {
//   "problem_summary": "texto...",
//   "reflection": "texto...",
//   "mood_before": "ansioso",
//   "mood_after": "más tranquilo",
//   "actions": [
//     {
//       "description": "Salir a caminar 10 minutos",
//       "status": "pending",
//       "notes": "Hacerlo hoy después de cenar"
//     }
//   ]
// }
//
// UserID and SessionID come in ToolContext.
func (t *JournalTool) Call(
	ctx context.Context,
	tctx ToolContext,
	input map[string]any,
) (map[string]any, error) {

	// Basic validation of context
	if tctx.UserID == "" || tctx.SessionID == "" {
		return nil, fmt.Errorf("journal_store: missing UserID or SessionID in ToolContext")
	}

	now := t.now()

	entry := &domain.JournalEntry{
		ID:            domain.JournalEntryID(generateID(now)),
		SessionID:     domain.SessionID(tctx.SessionID),
		UserID:        domain.UserID(tctx.UserID),
		CreatedAt:     now,
		UpdatedAt:     now,
		ProblemSummary: getString(input, "problem_summary"),
		Reflection:     getString(input, "reflection"),
		MoodBefore:     getString(input, "mood_before"),
		MoodAfter:      getString(input, "mood_after"),
		ActionPlan:     parseActions(input["actions"], now),
	}

	if err := t.store.AppendJournalEntry(entry); err != nil {
		return nil, fmt.Errorf("journal_store: append failed: %w", err)
	}

	return map[string]any{
		"status":       "ok",
		"entry_id":     string(entry.ID),
		"session_id":   string(entry.SessionID),
		"user_id":      string(entry.UserID),
		"created_at":   entry.CreatedAt,
		"actions_count": len(entry.ActionPlan),
	}, nil
}

// --- internal helpers --- //

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func parseActions(raw any, now time.Time) []domain.JournalAction {
	if raw == nil {
		return nil
	}

	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	var actions []domain.JournalAction
	for i, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}

		desc := getString(obj, "description")
		if desc == "" {
			continue
		}

		statusStr := getString(obj, "status")
		if statusStr == "" {
			statusStr = string(domain.ActionStatusPending)
		}
		status := domain.ActionStatus(statusStr)

		notes := getString(obj, "notes")

		actions = append(actions, domain.JournalAction{
			ID:          fmt.Sprintf("a-%d-%d", now.UnixNano(), i),
			Description: desc,
			Status:      status,
			Notes:       notes,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return actions
}

func generateID(now time.Time) string {
	return now.Format("20060102150405.000000000")
}
