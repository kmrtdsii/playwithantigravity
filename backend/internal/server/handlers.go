package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	s.Mux.HandleFunc("/api/remote/state", s.handleGetRemoteState)
	// s.Mux.HandleFunc("/api/sandbox/fork", s.handleForkSession) // REMOVED
	s.Mux.HandleFunc("/api/strategies", s.handleGetStrategies)
	s.Mux.HandleFunc("/api/remote/ingest", s.handleIngestRemote)
	s.Mux.HandleFunc("/api/remote/simulate-commit", s.handleSimulateRemoteCommit)
	s.Mux.HandleFunc("/api/remote/pull-requests", s.handleGetPullRequests)
	s.Mux.HandleFunc("/api/remote/pull-requests/create", s.handleCreatePullRequest)
	s.Mux.HandleFunc("/api/remote/pull-requests/merge", s.handleMergePullRequest)
	s.Mux.HandleFunc("/api/remote/reset", s.handleResetRemote)
}

func (s *Server) handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := git.GetBranchingStrategies()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
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
	// Generate complex ID
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

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

func (s *Server) handleGetRemoteState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	repo, ok := s.SessionManager.GetSharedRemote(name)

	if !ok {
		// Return empty/uninitialized state instead of 404 to avoid frontend crash?
		// Or 404. 404 is cleaner.
		http.Error(w, "remote not found", http.StatusNotFound)
		return
	}

	// Build state from the shared repo
	state := git.BuildGraphState(repo)
	// Add logic to populate shared remotes
	state.SharedRemotes = []string{name} // The requested one is definitely there.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (s *Server) handleSimulateRemoteCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		req.Message = "Simulated commit from team member"
	}

	if err := s.SessionManager.SimulateCommit(req.Name, req.Message); err != nil {
		http.Error(w, fmt.Sprintf("failed to simulate commit: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleIngestRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.SessionManager.IngestRemote(req.Name, req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetPullRequests(w http.ResponseWriter, r *http.Request) {
	prs := s.SessionManager.GetPullRequests()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prs)
}

func (s *Server) handleCreatePullRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		SourceBranch string `json:"sourceBranch"`
		TargetBranch string `json:"targetBranch"`
		Creator      string `json:"creator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pr, err := s.SessionManager.CreatePullRequest(req.Title, req.Description, req.SourceBranch, req.TargetBranch, req.Creator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pr)
}

func (s *Server) handleMergePullRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID         int    `json:"id"`
		RemoteName string `json:"remoteName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.SessionManager.MergePullRequest(req.ID, req.RemoteName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleResetRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "origin" // Default remote name
	}
	if err := s.SessionManager.RemoveRemote(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
