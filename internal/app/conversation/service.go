package conversation

import (
	"context"
	"time"

	"github.com/PabloGalante/farum-agent/internal/domain"
)

type Service struct {
	llm          domain.LLMClient
	sessionStore domain.SessionStore
	messageStore domain.MessageStore
	now          func() time.Time
}

func NewService(
	llm domain.LLMClient,
	sessionStore domain.SessionStore,
	messageStore domain.MessageStore,
) *Service {
	return &Service{
		llm:          llm,
		sessionStore: sessionStore,
		messageStore: messageStore,
		now:          time.Now,
	}
}

type StartSessionInput struct {
	UserID        domain.UserID
	PreferredMode domain.InteractionMode
	Title         string
}

type StartSessionOutput struct {
	Session *domain.Session
}

func (s *Service) StartSession(ctx context.Context, in StartSessionInput) (*StartSessionOutput, error) {
	now := s.now()

	session := &domain.Session{
		ID:            domain.SessionID(generateID()),
		UserID:        in.UserID,
		CreatedAt:     now,
		UpdatedAt:     now,
		PreferredMode: in.PreferredMode,
		Title:         in.Title,
	}

	if err := s.sessionStore.CreateSession(session); err != nil {
		return nil, err
	}

	// Optional: Welcome message from the agent
	welcome := &domain.Message{
		ID:        domain.MessageID(generateID()),
		SessionID: session.ID,
		Author:    domain.RoleAgent,
		Text:      "Hola, soy Farum. Qué te gustaría trabajar hoy?",
		CreatedAt: now,
		Mode:      session.PreferredMode,
	}

	if err := s.messageStore.AppendMessage(welcome); err != nil {
		return nil, err
	}

	return &StartSessionOutput{
		Session: session,
	}, nil
}

type SendMessageInput struct {
	SessionID domain.SessionID
	UserID    domain.UserID
	Text      string
}

type SendMessageOutput struct {
	UserMessage  *domain.Message
	AgentMessage *domain.Message
}

func (s *Service) SendMessage(ctx context.Context, in SendMessageInput) (*SendMessageOutput, error) {
	session, err := s.sessionStore.GetSession(in.SessionID)
	if err != nil {
		return nil, err
	}

	now := s.now()

	userMsg := &domain.Message{
		ID:        domain.MessageID(generateID()),
		SessionID: session.ID,
		Author:    domain.RoleUser,
		Text:      in.Text,
		CreatedAt: now,
		Mode:      session.PreferredMode,
	}

	if err := s.messageStore.AppendMessage(userMsg); err != nil {
		return nil, err
	}

	history, err := s.messageStore.GetMessagesBySession(session.ID, 20)
	if err != nil {
		return nil, err
	}

	replyText, err := s.llm.GenerateReply(in.Text, domain.ConversationContext{
		SessionID: session.ID,
		UserID:    in.UserID,
		Mode:      session.PreferredMode,
		History:   history,
	})
	if err != nil {
		return nil, err
	}

	agentMsg := &domain.Message{
		ID:        domain.MessageID(generateID()),
		SessionID: session.ID,
		Author:    domain.RoleAgent,
		Text:      replyText,
		CreatedAt: s.now(),
		Mode:      session.PreferredMode,
	}

	if err := s.messageStore.AppendMessage(agentMsg); err != nil {
		return nil, err
	}

	session.UpdatedAt = s.now()
	if err := s.sessionStore.UpdateSession(session); err != nil {
		return nil, err
	}

	return &SendMessageOutput{
		UserMessage:  userMsg,
		AgentMessage: agentMsg,
	}, nil
}

func (s *Service) GetSessionTimeline(
	ctx context.Context,
	sessionID domain.SessionID,
	limit int,
) (*domain.Session, []*domain.Message, error) {

	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		return nil, nil, err
	}

	msgs, err := s.messageStore.GetMessagesBySession(sessionID, limit)
	if err != nil {
		return nil, nil, err
	}

	return session, msgs, nil
}

// TODO: replace with something like UUID
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}
