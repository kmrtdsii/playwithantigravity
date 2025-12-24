package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kurobon/gitgym/backend/internal/git"
)

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
	session, ok := s.SessionManager.GetSession(req.SessionID)
	if !ok {
		log.Printf("Session %s not found (likely backend restart). Recreating...", req.SessionID)
		var err error
		session, err = s.SessionManager.CreateSession(req.SessionID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to restore session: " + err.Error()})
			return
		}
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
		if err.Error() == "session not found" {
			// Auto-restore session for graph view as well
			s.SessionManager.CreateSession(sessionID)
			state, err = s.SessionManager.GetGraphState(sessionID, showAll)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}
