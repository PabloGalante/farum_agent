package main

import (
	"context"
	"log"
	"net/http"
	"os"

	httpadapter "github.com/PabloGalante/farum-agent/internal/adapters/http"
	"github.com/PabloGalante/farum-agent/internal/adapters/llm"
	"github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/domain"
)

func main() {
	ctx := context.Background()

	// Choose between mock and Vertex by ENV (useful for dev)
	useMock := os.Getenv("FARUM_USE_MOCK_LLM") == "1"

	var (
		llmClient domain.LLMClient
		err       error
	)

	if useMock {
		log.Println("[LLM] Using MOCK LLM client")
		llmClient = llm.NewMockLLM()
	} else {
		log.Println("[LLM] Using Vertex LLM client")
		llmClient, err = llm.NewVertexClient(ctx)
		if err != nil {
			log.Fatalf("error initializing Vertex LLM client: %v", err)
		}
	}

	sessionStore := memory.NewSessionStore()
	messageStore := memory.NewMessageStore()

	svc := conversation.NewService(llmClient, sessionStore, messageStore)

	handler := httpadapter.NewServer(svc)

	port := getEnv("PORT", "8080")
	log.Println("Farum API listening on port:", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
