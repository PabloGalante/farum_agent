package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/PabloGalante/farum-agent/internal/app/conversation"
	"github.com/PabloGalante/farum-agent/internal/domain"
)

type Server struct {
	svc *conversation.Service
}

func NewServer(svc *conversation.Service) http.Handler {
	s := &Server{svc: svc}
	mux := http.NewServeMux()

	// /sessions → create session (POST)
	mux.HandleFunc("/sessions", s.handleSessions)

	// /sessions/{id}         →  GET: get session + messages
	// /sessions/{id}/messages → POST: send message
	mux.HandleFunc("/sessions/", s.handleSessionWithID)

	return mux
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

	out, err := s.svc.StartSession(
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
	_, msgs, err := s.svc.GetSessionTimeline(r.Context(), out.Session.ID, 5)
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
	session, msgs, err := s.svc.GetSessionTimeline(r.Context(), id, 0)
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

	out, err := s.svc.SendMessage(
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
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error": "internal server error",
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": "method not allowed",
	})
}
