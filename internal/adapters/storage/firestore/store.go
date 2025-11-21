package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

type Store struct {
	client *firestore.Client
}

// NewStore creates a Firestore store.
// Uses the project passed (FARUM_GCP_PROJECT).
func NewStore(ctx context.Context, projectID string) (*Store, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required for Firestore store")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("creating firestore client: %w", err)
	}

	return &Store{client: client}, nil
}

// ─────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────

func (s *Store) sessionsCol() *firestore.CollectionRef {
	return s.client.Collection("sessions")
}

func (s *Store) sessionDoc(id domain.SessionID) *firestore.DocumentRef {
	return s.sessionsCol().Doc(string(id))
}

func (s *Store) messagesCol(sessionID domain.SessionID) *firestore.CollectionRef {
	return s.sessionDoc(sessionID).Collection("messages")
}

func (s *Store) messageDoc(sessionID domain.SessionID, msgID domain.MessageID) *firestore.DocumentRef {
	return s.messagesCol(sessionID).Doc(string(msgID))
}

// ─────────────────────────────────────────
// Firestore Types
// ─────────────────────────────────────────

type sessionDoc struct {
	UserID        string    `firestore:"user_id"`
	Title         string    `firestore:"title"`
	PreferredMode string    `firestore:"preferred_mode"`
	CreatedAt     time.Time `firestore:"created_at"`
	UpdatedAt     time.Time `firestore:"updated_at"`
}

type messageDoc struct {
	SessionID   string    `firestore:"session_id"`
	Author      string    `firestore:"author"`
	Text        string    `firestore:"text"`
	Mode        string    `firestore:"mode"`
	CreatedAt   time.Time `firestore:"created_at"`
	Tags        []string  `firestore:"tags"`
	ReplyTo     *string   `firestore:"reply_to"`
	ContentType string    `firestore:"content_type"`
}

// ─────────────────────────────────────────
// SessionStore implementation
// ─────────────────────────────────────────

func (s *Store) CreateSession(session *domain.Session) error {
	ctx := context.Background()

	doc := sessionDoc{
		UserID:        string(session.UserID),
		Title:         session.Title,
		PreferredMode: string(session.PreferredMode),
		CreatedAt:     session.CreatedAt,
		UpdatedAt:     session.UpdatedAt,
	}

	_, err := s.sessionDoc(session.ID).Create(ctx, doc)
	if err != nil {
		return fmt.Errorf("firestore CreateSession: %w", err)
	}
	return nil
}

func (s *Store) UpdateSession(session *domain.Session) error {
	ctx := context.Background()

	doc := map[string]interface{}{
		"user_id":        string(session.UserID),
		"title":          session.Title,
		"preferred_mode": string(session.PreferredMode),
		"created_at":     session.CreatedAt,
		"updated_at":     session.UpdatedAt,
	}

	_, err := s.sessionDoc(session.ID).Set(ctx, doc, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("firestore UpdateSession: %w", err)
	}
	return nil
}

func (s *Store) GetSession(id domain.SessionID) (*domain.Session, error) {
	ctx := context.Background()

	snap, err := s.sessionDoc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("firestore GetSession: %w", err)
	}

	var doc sessionDoc
	if err := snap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("firestore GetSession decode: %w", err)
	}

	return &domain.Session{
		ID:            id,
		UserID:        domain.UserID(doc.UserID),
		Title:         doc.Title,
		PreferredMode: domain.InteractionMode(doc.PreferredMode),
		CreatedAt:     doc.CreatedAt,
		UpdatedAt:     doc.UpdatedAt,
	}, nil
}

func (s *Store) ListSessionsByUser(userID domain.UserID, limit int) ([]*domain.Session, error) {
	ctx := context.Background()

	q := s.sessionsCol().Where("user_id", "==", string(userID)).OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var out []*domain.Session
	for {
		snap, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("firestore ListSessionsByUser: %w", err)
		}

		var doc sessionDoc
		if err := snap.DataTo(&doc); err != nil {
			return nil, fmt.Errorf("decode sessionDoc: %w", err)
		}

		out = append(out, &domain.Session{
			ID:            domain.SessionID(snap.Ref.ID),
			UserID:        domain.UserID(doc.UserID),
			Title:         doc.Title,
			PreferredMode: domain.InteractionMode(doc.PreferredMode),
			CreatedAt:     doc.CreatedAt,
			UpdatedAt:     doc.UpdatedAt,
		})
	}
	return out, nil
}

// ─────────────────────────────────────────
// MessageStore implementation
// ─────────────────────────────────────────

func (s *Store) AppendMessage(msg *domain.Message) error {
	ctx := context.Background()

	var replyTo *string
	if msg.ReplyTo != nil {
		v := string(*msg.ReplyTo)
		replyTo = &v
	}

	doc := messageDoc{
		SessionID:   string(msg.SessionID),
		Author:      string(msg.Author),
		Text:        msg.Text,
		Mode:        string(msg.Mode),
		CreatedAt:   msg.CreatedAt,
		Tags:        msg.Tags,
		ReplyTo:     replyTo,
		ContentType: msg.ContentType,
	}

	_, err := s.messageDoc(msg.SessionID, msg.ID).Set(ctx, doc)
	if err != nil {
		return fmt.Errorf("firestore AppendMessage: %w", err)
	}
	return nil
}

func (s *Store) GetMessagesBySession(sessionID domain.SessionID, limit int) ([]*domain.Message, error) {
	ctx := context.Background()

	q := s.messagesCol(sessionID).OrderBy("created_at", firestore.Asc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var out []*domain.Message
	for {
		snap, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, fmt.Errorf("firestore GetMessagesBySession: %w", err)
		}

		var doc messageDoc
		if err := snap.DataTo(&doc); err != nil {
			return nil, fmt.Errorf("decode messageDoc: %w", err)
		}

		var replyTo *domain.MessageID
		if doc.ReplyTo != nil {
			id := domain.MessageID(*doc.ReplyTo)
			replyTo = &id
		}

		out = append(out, &domain.Message{
			ID:          domain.MessageID(snap.Ref.ID),
			SessionID:   sessionID,
			Author:      domain.Role(doc.Author),
			Text:        doc.Text,
			Mode:        domain.InteractionMode(doc.Mode),
			CreatedAt:   doc.CreatedAt,
			Tags:        doc.Tags,
			ReplyTo:     replyTo,
			ContentType: doc.ContentType,
		})
	}
	return out, nil
}
