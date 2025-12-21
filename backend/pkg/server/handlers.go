package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
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

	parts := strings.Fields(req.Command)
	
	// Handle Shortcuts
	if len(parts) > 0 {
		switch parts[0] {
		case "reset":
			newParts := []string{"git", "reset"}
			parts = append(newParts, parts[1:]...)
		case "add":
			newParts := []string{"git", "add"}
			parts = append(newParts, parts[1:]...)
		case "commit":
			newParts := []string{"git", "commit", "-m"}
			parts = append(newParts, parts[1:]...)
		case "merge":
			newParts := []string{"git", "merge"}
			parts = append(newParts, parts[1:]...)
		case "tag":
			newParts := []string{"git", "tag"}
			parts = append(newParts, parts[1:]...)
		case "rebase":
			newParts := []string{"git", "rebase"}
			parts = append(newParts, parts[1:]...)
		case "checkout":
			newParts := []string{"git", "checkout"}
			parts = append(newParts, parts[1:]...)
		case "branch":
			newParts := []string{"git", "branch"}
			parts = append(newParts, parts[1:]...)
		case "switch":
			newParts := []string{"git", "switch"}
			parts = append(newParts, parts[1:]...)
		}
	}

	log.Printf("Command received: user=%s cmd=%s parts=%v", req.SessionID, req.Command, parts)
	
	if len(parts) > 0 {
		if parts[0] == "git" {
			session, err := s.SessionManager.GetSession(req.SessionID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			
			cmdName := ""
			args := []string{}
			if len(parts) > 1 {
				cmdName = parts[1]
				args = parts[1:]
			}
			
			output, err := git.Dispatch(r.Context(), session, cmdName, args)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"output": output})
			return

		} else if parts[0] == "touch" {
			if len(parts) < 2 {
				http.Error(w, "Filename required", http.StatusBadRequest)
				return
			}
			err := s.SessionManager.TouchFile(req.SessionID, parts[1])
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"output": "File updated"})
			return

		} else if parts[0] == "ls" {
			output, err := s.SessionManager.ListFiles(req.SessionID)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"output": output})
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": "Only git commands supported right now"})
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
