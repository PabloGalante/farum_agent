package main

import (
	"context"
	"log"
	"net/http"
	"time"

	httpadapter "github.com/PabloGalante/farum-agent/internal/adapters/http"
	llmadapter "github.com/PabloGalante/farum-agent/internal/adapters/llm"
	firestorestore "github.com/PabloGalante/farum-agent/internal/adapters/storage/firestore"
	memstore "github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/app/tools"
	"github.com/PabloGalante/farum-agent/internal/config"
	"github.com/PabloGalante/farum-agent/internal/domain"
	"github.com/PabloGalante/farum-agent/internal/observability"
)

func main() {
	ctx := context.Background()

	// 1) Load centralized configuration
	cfg := config.Load()

	logger := observability.Logger()
	logger.Info("starting Farum",
		"mode", cfg.Mode,
		"port", cfg.Port,
		"storage_backend", cfg.StorageBackend,
		"use_mock_llm", cfg.UseMockLLM,
	)

	// 2) Create LLMClient according to config
	var (
		llmClient domain.LLMClient
		err       error
	)

	if cfg.UseMockLLM {
		logger.Info("[LLM] Using MOCK LLM client")
		llmClient = llmadapter.NewMockLLM()
	} else {
		logger.Info("[LLM] Using Vertex LLM client",
			"project", cfg.GCPProjectID,
			"location", cfg.GCPLocation,
			"model", cfg.ModelName,
		)

		llmClient, err = llmadapter.NewVertexClient(ctx, llmadapter.VertexConfig{
			ProjectID: cfg.GCPProjectID,
			Location:  cfg.GCPLocation,
			ModelName: cfg.ModelName,
		})
		if err != nil {
			logger.Error("error initializing Vertex LLM client", "error", err)
			log.Fatal(err)
		}
	}

	// 3) Storage: Firestore or Memory according to config.StorageBackend
	var sessionStore domain.SessionStore
	var messageStore domain.MessageStore
	var journalStore domain.JournalStore

	switch cfg.StorageBackend {
	case "firestore":
		if cfg.GCPProjectID == "" {
			logger.Error("FARUM_GCP_PROJECT is required for Firestore storage backend", "backend", "firestore")
			log.Fatal("FARUM_GCP_PROJECT is required for Firestore storage backend")
		}

		logger.Info("[STORE] Using Firestore storage", "project", cfg.GCPProjectID)
		fsStore, err := firestorestore.NewStore(ctx, cfg.GCPProjectID)
		if err != nil {
			logger.Error("error initializing Firestore store", "error", err)
			log.Fatal(err)
		}

		// 1 store, implements 3 interfaces
		sessionStore = fsStore
		messageStore = fsStore
		journalStore = nil // TODO

	default:
		logger.Info("[STORE] Using in-memory storage", "backend", "memory")
		sessionStore = memstore.NewSessionStore()
		messageStore = memstore.NewMessageStore()
		journalStore = memstore.NewJournalStore()
	}

	var journalTool *tools.JournalTool
	if journalStore != nil {
		journalTool = tools.NewJournalTool(journalStore)
	}

	// 4) Conversation Service
	svc := conversation.NewService(llmClient, sessionStore, messageStore, journalTool)

	// 5) HTTP server
	handler := httpadapter.NewServer(svc)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("Farum API listening", "port", cfg.Port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("HTTP server error", "error", err)
		log.Fatal(err)
	}
}
