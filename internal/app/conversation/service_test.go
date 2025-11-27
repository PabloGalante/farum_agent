package conversation_test

import (
	"context"
	"testing"

	"github.com/PabloGalante/farum-agent/internal/adapters/llm"
	"github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/domain"
)

func TestStartSessionAndSendMessage(t *testing.T) {
	ctx := context.Background()

	llmClient := llm.NewMockLLM()
	sessionStore := memory.NewSessionStore()
	messageStore := memory.NewMessageStore()

	svc := conversation.NewService(llmClient, sessionStore, messageStore, nil)

	out, err := svc.StartSession(ctx, conversation.StartSessionInput{
		UserID:        domain.UserID("test-user"),
		PreferredMode: domain.ModeCheckIn,
		Title:         "Test session",
	})
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if out.Session.ID == "" {
		t.Fatalf("expected session id, got empty")
	}

	reply, err := svc.SendMessage(ctx, conversation.SendMessageInput{
		SessionID: out.Session.ID,
		UserID:    out.Session.UserID,
		Text:      "Hola Farum",
	})
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if reply.AgentMessage == nil || reply.AgentMessage.Text == "" {
		t.Fatalf("expected non-empty agent reply")
	}
}
