package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	// Remote View: We generally want to see everything reachable from heads/tags.
	// Passing true (ShowAll) ensures we see everything if BFS misses something,
	// but strictly BFS from refs (false) is cleaner for "reachable".
	// However, to debug "missing tags", let's enable ShowAll=true for Remote View.
	stateObj := state.BuildGraphState(repo, true)
	// Add logic to populate shared remotes
	stateObj.SharedRemotes = []string{name} // The requested one is definitely there.

	// CLEANUP FOR VISUALIZATION:
	// The "Remote View" represents the server state.
	// In our simulated backend, the server ("origin") has branches which are stored as
	// "refs/remotes/origin/..." because we cloned them.
	// We should display these as "Branches" to the user, not "Remote Branches".

	for name, sha := range stateObj.RemoteBranches {
		// name is like "origin/main", "origin/feature"
		// We want to show "main", "feature"
		// Simple heuristic: strip "origin/" prefix.
		if len(name) > 7 && name[:7] == "origin/" {
			branchName := name[7:]
			// Avoid overwriting if we already have it (e.g. main matches refs/heads/main)
			if _, exists := stateObj.Branches[branchName]; !exists {
				stateObj.Branches[branchName] = sha
			}
		}
	}

	// Only show local branches (simulated as server branches) and tags.
	// stateObj.Remotes = []state.Remote{}               // Do not clear.
	stateObj.RemoteBranches = make(map[string]string) // Clear remote tracking branches

	// If no remotes (created bare repo), inject self as origin for UI display
	// [FIX] Do NOT auto-inject 'origin' with pseudo-URL. This confuses users into thinking
	// 'git remote add' succeeded with a default URL.
	// if len(stateObj.Remotes) == 0 {
	// 	stateObj.Remotes = []state.Remote{
	// 		{
	// 			Name: "origin", // UI expects 'origin' or first remote
	// 			URLs: []string{fmt.Sprintf("remote://gitgym/%s.git", name)},
	// 		},
	// 	}
	// }

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stateObj)
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleIngestRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Depth int    `json:"depth"` // Optional: 0 means full clone
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Propagate Context
	if err := s.SessionManager.IngestRemote(r.Context(), req.Name, req.URL, req.Depth); err != nil {
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
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(estimate)
}

func (s *Server) handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := state.GetBranchingStrategies()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(strategies)
}

// CreateRemoteRequest structure
type CreateRemoteRequest struct {
	Name string `json:"name"`
}

// handleCreateRemote creates a new bare repository
func (s *Server) handleCreateRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Get Session ID
	// Priority: Header > Cookie
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		cookie, err := r.Cookie("session_id")
		if err == nil {
			sessionID = cookie.Value
		}
	}

	if sessionID == "" {
		http.Error(w, "Session ID required (X-Session-ID header or session_id cookie)", http.StatusBadRequest)
		return
	}

	var req CreateRemoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	// 2. Create Repository
	if err := s.SessionManager.CreateBareRepository(r.Context(), sessionID, req.Name); err != nil {
		if err.Error() == "invalid repository name: only alphanumeric, hyphen and underscore allowed" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Differentiate error types if possible, but 500 is safe for now
		http.Error(w, fmt.Sprintf("Failed to create repository: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Return Success
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":   "Repository created successfully",
		"name":      req.Name,
		"remoteUrl": fmt.Sprintf("remote://gitgym/%s.git", req.Name),
	})
}

// handleListRemotes returns the list of currently registered shared remotes
func (s *Server) handleListRemotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get unique keys (filter out duplicates like path aliases)
	s.SessionManager.RLock()
	seen := make(map[string]bool)
	var names []string
	for key := range s.SessionManager.SharedRemotes {
		// Only include simple names (no paths, no URLs)
		if key != "" && key[0] != '/' && len(key) < 50 && key != "origin" {
			// Filter out keys that look like URLs or paths (contain : or /)
			if !strings.Contains(key, ":") && !strings.Contains(key, "/") {
				if _, dup := seen[key]; !dup {
					seen[key] = true
					names = append(names, key)
				}
			}
		}
	}
	s.SessionManager.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"remotes": names,
	})
}
