package httpadapter

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/app/journal"
	"github.com/PabloGalante/farum-agent/internal/domain"
)

type Server struct {
	convSvc    *conversation.Service
	journalSvc *journal.Service
}

func NewServer(convSvc *conversation.Service, journalSvc *journal.Service) http.Handler {
	s := &Server{
		convSvc:    convSvc,
		journalSvc: journalSvc,
	}
	mux := http.NewServeMux()

	// healthcheck
	mux.HandleFunc("/healthz", s.handleHealth)

	// /sessions → create session (POST)
	mux.HandleFunc("/sessions", s.handleSessions)

	// /sessions/{id}         →  GET: get session + messages
	// /sessions/{id}/messages → POST: send message
	mux.HandleFunc("/sessions/", s.handleSessionWithID)

	// /users/{id}/journal → GET: get user's journal entries
	mux.HandleFunc("/users/", s.handleUserWithID)

	return chainMiddlewares(mux, withCORS, withLogging)
}

// ─────────────────────────────────────────────
// DTOs (request/response)
// ─────────────────────────────────────────────

type createSessionRequest struct {
	UserID        string `json:"user_id"`
	PreferredMode string `json:"preferred_mode,omitempty"`
	Title         string `json:"title,omitempty"`
}

type createSessionResponse struct {
	Session sessionResponse  `json:"session"`
	Welcome *messageResponse `json:"welcome_message,omitempty"`
}

type sessionResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Title         string    `json:"title"`
	PreferredMode string    `json:"preferred_mode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type messageResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	Mode      string    `json:"mode"`
	CreatedAt time.Time `json:"created_at"`
}

type sendMessageRequest struct {
	UserID string `json:"user_id"`
	Text   string `json:"text"`
}

type sendMessageResponse struct {
	UserMessage  messageResponse `json:"user_message"`
	AgentMessage messageResponse `json:"agent_message"`
}

type getSessionResponse struct {
	Session  sessionResponse   `json:"session"`
	Messages []messageResponse `json:"messages"`
}

// Journal DTOs

type journalActionResponse struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type journalEntryResponse struct {
	ID             string                  `json:"id"`
	SessionID      string                  `json:"session_id"`
	UserID         string                  `json:"user_id"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
	ProblemSummary string                  `json:"problem_summary"`
	ActionPlan     []journalActionResponse `json:"action_plan"`
	Reflection     string                  `json:"reflection"`
	MoodBefore     string                  `json:"mood_before"`
	MoodAfter      string                  `json:"mood_after"`
}

// ─────────────────────────────────────────────
// Basic routing
// ─────────────────────────────────────────────

// /sessions
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateSession(w, r)
	default:
		methodNotAllowed(w)
	}
}

// /sessions/{id} or /sessions/{id}/messages
func (s *Server) handleSessionWithID(w http.ResponseWriter, r *http.Request) {
	// expected path:
	// /sessions/{id}
	// /sessions/{id}/messages
	path := strings.TrimPrefix(r.URL.Path, "/sessions/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(path, "/")
	id := parts[0]

	if id == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 {
		// /sessions/{id}
		switch r.Method {
		case http.MethodGet:
			s.handleGetSession(w, r, domain.SessionID(id))
		default:
			methodNotAllowed(w)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "messages" {
		// /sessions/{id}/messages
		switch r.Method {
		case http.MethodPost:
			s.handleSendMessage(w, r, domain.SessionID(id))
		default:
			methodNotAllowed(w)
		}
		return
	}

	http.NotFound(w, r)
}

// /users/{id}/journal
func (s *Server) handleUserWithID(w http.ResponseWriter, r *http.Request) {
	// expected path:
	// /users/{id}/journal
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(path, "/")
	userID := parts[0]

	if userID == "" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "journal" {
		switch r.Method {
		case http.MethodGet:
			s.handleGetUserJournal(w, r, domain.UserID(userID))
		default:
			methodNotAllowed(w)
		}
		return
	}

	http.NotFound(w, r)
}

// ─────────────────────────────────────────────
// Concrete handlers
// ─────────────────────────────────────────────

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid JSON body")
		return
	}

	if req.UserID == "" {
		badRequest(w, "user_id is required")
		return
	}

	mode := parseInteractionMode(req.PreferredMode)

	out, err := s.convSvc.StartSession(
		r.Context(),
		conversation.StartSessionInput{
			UserID:        domain.UserID(req.UserID),
			PreferredMode: mode,
			Title:         req.Title,
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}

	// To obtain welcome message, request the timeline limited to the 1-2 most recent messages.
	_, msgs, err := s.convSvc.GetSessionTimeline(r.Context(), out.Session.ID, 5)
	if err != nil {
		internalError(w, err)
		return
	}

	var welcome *messageResponse
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		if last.Author == domain.RoleAgent {
			m := toMessageResponse(last)
			welcome = &m
		}
	}

	resp := createSessionResponse{
		Session: toSessionResponse(out.Session),
		Welcome: welcome,
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request, id domain.SessionID) {
	session, msgs, err := s.convSvc.GetSessionTimeline(r.Context(), id, 0)
	if err != nil {
		// Any error is (for simplicity) → 404 if it is "not found"
		if errors.Is(err, errors.New("session not found")) {
			http.NotFound(w, r)
			return
		}
		internalError(w, err)
		return
	}

	resp := getSessionResponse{
		Session:  toSessionResponse(session),
		Messages: toMessagesResponse(msgs),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request, sessionID domain.SessionID) {
	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid JSON body")
		return
	}

	if req.UserID == "" {
		badRequest(w, "user_id is required")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		badRequest(w, "text is required")
		return
	}

	out, err := s.convSvc.SendMessage(
		r.Context(),
		conversation.SendMessageInput{
			SessionID: sessionID,
			UserID:    domain.UserID(req.UserID),
			Text:      req.Text,
		},
	)
	if err != nil {
		internalError(w, err)
		return
	}

	resp := sendMessageResponse{
		UserMessage:  toMessageResponse(out.UserMessage),
		AgentMessage: toMessageResponse(out.AgentMessage),
	}

	writeJSON(w, http.StatusOK, resp)
}

// GET /users/{id}/journal
func (s *Server) handleGetUserJournal(w http.ResponseWriter, r *http.Request, userID domain.UserID) {
	if s.journalSvc == nil {
		// Disabled journal (for now this could happen in GCP mode without FirestoreJournalStore)
		writeJSON(w, http.StatusOK, []journalEntryResponse{})
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	entries, err := s.journalSvc.GetUserJournal(r.Context(), userID, limit)
	if err != nil {
		internalError(w, err)
		return
	}

	resp := make([]journalEntryResponse, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, toJournalEntryResponse(e))
	}

	writeJSON(w, http.StatusOK, resp)
}

// ─────────────────────────────────────────────
// Conversation Helpers
// ─────────────────────────────────────────────

func toSessionResponse(s *domain.Session) sessionResponse {
	return sessionResponse{
		ID:            string(s.ID),
		UserID:        string(s.UserID),
		Title:         s.Title,
		PreferredMode: string(s.PreferredMode),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toMessageResponse(m *domain.Message) messageResponse {
	return messageResponse{
		ID:        string(m.ID),
		SessionID: string(m.SessionID),
		Author:    string(m.Author),
		Text:      m.Text,
		Mode:      string(m.Mode),
		CreatedAt: m.CreatedAt,
	}
}

func toMessagesResponse(msgs []*domain.Message) []messageResponse {
	out := make([]messageResponse, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toMessageResponse(m))
	}
	return out
}

func toJournalEntryResponse(e *domain.JournalEntry) journalEntryResponse {
	actions := make([]journalActionResponse, 0, len(e.ActionPlan))
	for _, a := range e.ActionPlan {
		actions = append(actions, journalActionResponse{
			ID:          a.ID,
			Description: a.Description,
			Status:      string(a.Status),
			Notes:       a.Notes,
			CreatedAt:   a.CreatedAt,
			UpdatedAt:   a.UpdatedAt,
		})
	}

	return journalEntryResponse{
		ID:             string(e.ID),
		SessionID:      string(e.SessionID),
		UserID:         string(e.UserID),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
		ProblemSummary: e.ProblemSummary,
		ActionPlan:     actions,
		Reflection:     e.Reflection,
		MoodBefore:     e.MoodBefore,
		MoodAfter:      e.MoodAfter,
	}
}

func parseInteractionMode(s string) domain.InteractionMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "check_in", "checkin":
		return domain.ModeCheckIn
	case "deep_dive", "deep":
		return domain.ModeDeepDive
	case "action_plan", "action":
		return domain.ModeActionPlan
	default:
		return domain.ModeCheckIn
	}
}

// ─────────────────────────────────────────────
// HTTP Helpers
// ─────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{
		"error": msg,
	})
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)

	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error": "internal server error",
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": "method not allowed",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
