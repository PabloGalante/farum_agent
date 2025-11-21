package main

import (
	"context"
	"log"
	"net/http"
	"os"

	httpadapter "github.com/PabloGalante/farum-agent/internal/adapters/http"
	"github.com/PabloGalante/farum-agent/internal/adapters/llm"
	firestorestore "github.com/PabloGalante/farum-agent/internal/adapters/storage/firestore"
	memstore "github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
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

	// Storage: Firestore or Memory
	storageBackend := getEnv("FARUM_STORAGE_BACKEND", "memory")

	var sessionStore domain.SessionStore
	var messageStore domain.MessageStore

	switch storageBackend {
	case "firestore":
		projectID := getEnv("FARUM_GCP_PROJECT", "")
		if projectID == "" {
			log.Fatal("FARUM_GCP_PROJECT is required for Firestore storage backend")
		}

		log.Printf("[STORE] Using Firestore storage (project=%s)", projectID)
		fsStore, err := firestorestore.NewStore(ctx, projectID)
		if err != nil {
			log.Fatalf("error initializing Firestore store: %v", err)
		}

		// 1 store, implements 2 interfaces
		sessionStore = fsStore
		messageStore = fsStore

	default:
		log.Println("[STORE] Using in-memory storage")
		sessionStore = memstore.NewSessionStore()
		messageStore = memstore.NewMessageStore()
	}

	// Conversation Service
	svc := conversation.NewService(llmClient, sessionStore, messageStore)

	// HTTP server
	handler := httpadapter.NewServer(svc)

	port := ":" + getEnv("PORT", "8080")
	log.Println("Farum API listening on port:", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
