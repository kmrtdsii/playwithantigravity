package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/kurobon/gitgym/backend/internal/state"
)

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
	stateObj := state.BuildGraphState(repo)
	// Add logic to populate shared remotes
	stateObj.SharedRemotes = []string{name} // The requested one is definitely there.

	// CLEANUP FOR VISUALIZATION:
	// Only show local branches (simulated as server branches) and tags.
	stateObj.Remotes = []state.Remote{}               // Clear remotes
	stateObj.RemoteBranches = make(map[string]string) // Clear remote tracking branches

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateObj)
}

func (s *Server) handleSimulateRemoteCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
		Author  string `json:"author"` // Optional
		Email   string `json:"email"`  // Optional
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

	// Resolve Session
	sessionID := "user-session-1"
	session, ok := s.SessionManager.GetSession(sessionID)
	if !ok {
		var err error
		session, err = s.SessionManager.CreateSession(sessionID)
		if err != nil {
			http.Error(w, "failed to create session: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Dispatch simulate-commit
	args := []string{"simulate-commit", req.Name, req.Message}
	if req.Author != "" && req.Email != "" {
		args = append(args, req.Author, req.Email)
	}

	_, err := git.Dispatch(r.Context(), session, "simulate-commit", args)

	if err != nil {
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
	// Propagate Context
	if err := s.SessionManager.IngestRemote(r.Context(), req.Name, req.URL); err != nil {
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

func (s *Server) handleGetRemoteInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	estimate, err := git.GetCloneEstimate(url)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estimate)
}

func (s *Server) handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := state.GetBranchingStrategies()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}
