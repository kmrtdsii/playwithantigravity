package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func (s *Server) handleGetPullRequests(w http.ResponseWriter, r *http.Request) {
	prs := s.SessionManager.GetPullRequests()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(prs)
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
	_ = json.NewEncoder(w).Encode(pr)
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

	// Resolve Session (Use Default "user-session-1" for now as explained)
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

	// Dispatch "merge-pr"
	// Output is ignored for now, checking error
	_, err := git.Dispatch(r.Context(), session, "merge-pr", []string{"merge-pr", fmt.Sprintf("%d", req.ID), req.RemoteName})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeletePullRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.SessionManager.DeletePullRequest(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
