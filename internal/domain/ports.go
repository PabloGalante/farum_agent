package domain

import "context"

// LLMClient defines how the core application interacts with an LLM service.
type LLMClient interface {
	GenerateReply(ctx context.Context, prompt string, convCtx ConversationContext) (string, error)
}

// ConversationContext gives the LLM minimal context about the conversation.
type ConversationContext struct {
	SessionID SessionID
	UserID    UserID
	Mode      InteractionMode
	History   []*Message // for the MVP, last N interactions
}

// SessionStore defines session's persistence
type SessionStore interface {
	CreateSession(session *Session) error
	UpdateSession(session *Session) error
	GetSession(id SessionID) (*Session, error)
	ListSessionsByUser(userID UserID, limit int) ([]*Session, error)
}

// MessageStore defines message's persistence
type MessageStore interface {
	AppendMessage(msg *Message) error
	GetMessagesBySession(sessionID SessionID, limit int) ([]*Message, error)
}
