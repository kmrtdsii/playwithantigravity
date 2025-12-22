package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

type Server struct {
	SessionManager *git.SessionManager
	Mux            *http.ServeMux
}

func NewServer(sm *git.SessionManager) *Server {
	s := &Server{
		SessionManager: sm,
		Mux:            http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Mux.HandleFunc("/ping", s.handlePing)
	s.Mux.HandleFunc("/api/session/init", s.handleInitSession)
	s.Mux.HandleFunc("/api/command", s.handleExecCommand)
	s.Mux.HandleFunc("/api/state", s.handleGetGraphState)
	s.Mux.HandleFunc("/api/sandbox/fork", s.handleForkSession)
	s.Mux.HandleFunc("/api/strategies", s.handleGetStrategies)
}

func (s *Server) handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := git.GetBranchingStrategies()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

func (s *Server) handleForkSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Request Body: { "source_id": "...", "target_id": "..." }
	type ForkRequest struct {
		SourceID string `json:"source_id"`
		TargetID string `json:"target_id"`
	}

	var req ForkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SourceID == "" || req.TargetID == "" {
		http.Error(w, "source_id and target_id required", http.StatusBadRequest)
		return
	}

	newSession, err := s.SessionManager.ForkSession(req.SourceID, req.TargetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "forked",
		"sessionId": newSession.ID,
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Mux.ServeHTTP(w, r)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "pong",
		"system":  "GitGym Backend (pkg/server)",
	})
}

func (s *Server) handleInitSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Generate simple ID
	sessionID := "user-session-1" // Fixed for now

	if _, err := s.SessionManager.CreateSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "session created",
		"sessionId": sessionID,
	})
}

type CommandRequest struct {
	SessionID string `json:"sessionId"`
	Command   string `json:"command"`
}

func (s *Server) handleExecCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = "user-session-1" // Default for testing
	}

	// Logging
	log.Printf("Command received: user=%s cmd=%s", req.SessionID, req.Command)

	// 1. Parse Command & Resolve Aliases
	cmdName, args := git.ParseCommand(req.Command)
	if cmdName == "" {
		// Empty command
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"output": ""})
		return
	}

	// 2. Get Session
	session, err := s.SessionManager.GetSession(req.SessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 3. Dispatch Command
	// This now handles 'touch', 'ls', 'cd', 'rm' and all 'git' commands uniformly
	output, err := git.Dispatch(r.Context(), session, cmdName, args)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

func (s *Server) handleGetGraphState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = "user-session-1" // Default
	}

	showAll := r.URL.Query().Get("showAll") == "true"

	state, err := s.SessionManager.GetGraphState(sessionID, showAll)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}
