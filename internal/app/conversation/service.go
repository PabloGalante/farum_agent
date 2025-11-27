package conversation

import (
	"context"
	"time"

	"github.com/PabloGalante/farum-agent/internal/app/agentflow"
	"github.com/PabloGalante/farum-agent/internal/app/tools"
	"github.com/PabloGalante/farum-agent/internal/domain"
	"github.com/PabloGalante/farum-agent/internal/observability"
)

type Service struct {
	llm          domain.LLMClient
	sessionStore domain.SessionStore
	messageStore domain.MessageStore
	now          func() time.Time

	journalTool  *tools.JournalTool
	orchestrator *agentflow.Orchestrator
}

func NewService(
	llm domain.LLMClient,
	sessionStore domain.SessionStore,
	messageStore domain.MessageStore,
	journalTool *tools.JournalTool,
) *Service {
	var toolForOrchestrator tools.Tool
	if journalTool != nil {
		toolForOrchestrator = journalTool
	}

	return &Service{
		llm:          llm,
		sessionStore: sessionStore,
		messageStore: messageStore,
		now:          time.Now,
		journalTool:  journalTool,
		orchestrator: agentflow.NewDefaultOrchestrator(llm, toolForOrchestrator),
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

	log := observability.LoggerFromContext(ctx).With(
		"user_id", in.UserID,
		"preferred_mode", in.PreferredMode,
	)
	log.Info("starting new session")


	session := &domain.Session{
		ID:            domain.SessionID(generateID()),
		UserID:        in.UserID,
		CreatedAt:     now,
		UpdatedAt:     now,
		PreferredMode: in.PreferredMode,
		Title:         in.Title,
	}

	if err := s.sessionStore.CreateSession(session); err != nil {
		log.Error("failed to create session", "error", err)
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
		log.Error("failed to append welcome message", "error", err)
		return nil, err
	}

	log.Info("session started", "session_id", session.ID)

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

	log := observability.LoggerFromContext(ctx).With(
		"session_id", session.ID,
		"user_id", session.UserID,
		"mode", session.PreferredMode,
	)
	log.Info("sending message", "text", in.Text)

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
		log.Error("failed to append user message", "error", err)
		return nil, err
	}

	history, err := s.messageStore.GetMessagesBySession(session.ID, 20)
	if err != nil {
		log.Error("failed to load history", "error", err)
		return nil, err
	}

	convCtx := domain.ConversationContext{
		SessionID: session.ID,
		UserID:    session.UserID,
		Mode:      session.PreferredMode,
		History:   history,
	}

	replyText, err := s.orchestrator.Run(ctx, in.Text, convCtx)
	if err != nil {
		log.Error("orchestrator failed", "error", err)
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
		log.Error("failed to append agent message", "error", err)
		return nil, err
	}

	session.UpdatedAt = s.now()
	if err := s.sessionStore.UpdateSession(session); err != nil {
		log.Error("failed to update session", "error", err)
		return nil, err
	}

	log.Info("send message completed")

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

	log := observability.LoggerFromContext(ctx).With(
		"session_id", sessionID,
		"limit", limit,
	)

	session, err := s.sessionStore.GetSession(sessionID)
	if err != nil {
		log.Error("failed to get session", "error", err)
		return nil, nil, err
	}

	msgs, err := s.messageStore.GetMessagesBySession(sessionID, limit)
	if err != nil {
		log.Error("failed to get messages", "error", err)
		return nil, nil, err
	}

	log.Info("fetched session timeline", "message_count", len(msgs))

	return session, msgs, nil
}

// TODO: replace with something like UUID
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}
