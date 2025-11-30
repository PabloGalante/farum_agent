package httpadapter_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpadapter "github.com/PabloGalante/farum-agent/internal/adapters/http"
	"github.com/PabloGalante/farum-agent/internal/adapters/llm"
	"github.com/PabloGalante/farum-agent/internal/adapters/storage/memory"
	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	journalapp "github.com/PabloGalante/farum-agent/internal/app/journal"
	"github.com/PabloGalante/farum-agent/internal/app/tools"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	llmClient := llm.NewMockLLM()
	sessionStore := memory.NewSessionStore()
	messageStore := memory.NewMessageStore()
	journalStore := memory.NewJournalStore()

	var journalTool *tools.JournalTool
	if journalStore != nil {
		journalTool = tools.NewJournalTool(journalStore)
	}

	convSvc := conversation.NewService(llmClient, sessionStore, messageStore, journalTool)
	journalSvc := journalapp.NewService(journalStore)

	return httpadapter.NewServer(convSvc, journalSvc)
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCreateSessionAndSendMessage(t *testing.T) {
	srv := newTestServer(t)

	// Create session
	body := []byte(`{"user_id":"test-user","preferred_mode":"check_in","title":"Test"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", w.Code, w.Body.String())
	}
}
