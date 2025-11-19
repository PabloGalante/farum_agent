package main

import (
	"context"
	"fmt"

	"github.com/PabloGalante/farum-agent/internal/adapters/llm"
	"github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/domain"
)

func main() {
	llmClient := llm.NewMockLLM()
	sessionStore := memory.NewSessionStore()
	messageStore := memory.NewMessageStore()

	svc := conversation.NewService(llmClient, sessionStore, messageStore)

	ctx := context.Background()

	out, err := svc.StartSession(ctx, conversation.StartSessionInput{
		UserID:        domain.UserID("user-123"),
		PreferredMode: domain.ModeCheckIn,
		Title:         "Primera sesión con Farum",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Session creada:", out.Session.ID)

	reply, err := svc.SendMessage(ctx, conversation.SendMessageInput{
		SessionID: out.Session.ID,
		UserID:    out.Session.UserID,
		Text:      "Estoy un poco abrumado con el laburo y la facu.",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Usuario:", reply.UserMessage.Text)
	fmt.Println("Farum :", reply.AgentMessage.Text)
}
