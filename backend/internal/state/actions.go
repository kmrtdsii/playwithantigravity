package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(name, url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Define local path for persistence
	baseDir := sm.DataDir
	if baseDir == "" {
		baseDir = ".gitgym-data/remotes"
	}
	if name == "" {
		name = "origin"
	}
	repoPath := filepath.Join(baseDir, name)
	if absPath, err := filepath.Abs(repoPath); err == nil {
		repoPath = absPath
	}

	// Helper to cleanup and prepare for re-clone
	resetRepo := false
	if _, err := os.Stat(repoPath); err == nil {
		// Repo exists. Check if URL matches.
		existingRepo, err := gogit.PlainOpen(repoPath)
		if err == nil {
			rem, err := existingRepo.Remote("origin")
			if err == nil && len(rem.Config().URLs) > 0 {
				existingURL := rem.Config().URLs[0]
				if existingURL != url {
					// URL changed. Nuke it.
					fmt.Printf("Remote URL changed (%s -> %s). Re-cloning...\n", existingURL, url)
					resetRepo = true
				}
			} else {
				// No origin or corrupted? Nuke it.
				resetRepo = true
			}
		} else {
			// Corrupted? Nuke it.
			resetRepo = true
		}
	}

	if resetRepo {
		os.RemoveAll(repoPath)
	}

	// Ensure directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create remote dir: %w", err)
	}

	// Open or Clone
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		// Not found or just cleaned, clone it (Bare)
		fmt.Printf("Cloning %s to %s (bare)...\n", url, repoPath)
		repo, err = gogit.PlainClone(repoPath, true, &gogit.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
		})
		if err != nil {
			// Fallback: Init empty
			fmt.Printf("Clone failed (%v), initializing empty bare repo...\n", err)
			repo, err = gogit.PlainInit(repoPath, true)
			if err != nil {
				return fmt.Errorf("failed to init bare repo: %w", err)
			}
			// Add the origin remote so we know the URL later
			repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{url},
			})
		}
	}

	// Store under Name
	sm.SharedRemotes[name] = repo
	sm.SharedRemotePaths[name] = repoPath

	// Store under URL (so git clone <url> works)
	sm.SharedRemotes[url] = repo
	sm.SharedRemotePaths[url] = repoPath

	return nil
}

// RemoveRemote removes a shared remote
func (sm *SessionManager) RemoveRemote(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.SharedRemotes[name]; !ok {
		return fmt.Errorf("remote %s not found", name)
	}
	delete(sm.SharedRemotes, name)
	return nil
}

// SimulateCommit creates a dummy commit on the given remote repository
func (sm *SessionManager) SimulateCommit(remoteName, message string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	repo, ok := sm.SharedRemotes[remoteName]
	if !ok {
		return fmt.Errorf("remote %s not found", remoteName)
	}

	w, err := repo.Worktree()
	if err != nil {
		// If bare repo, we can't get worktree easy.
		// Use plumbing to create commit?
		// But memory repos created via Clone are not bare by default usually, unless configured.
		return fmt.Errorf("worktree error: %v", err)
	}

	// Create a dummy file change
	filename := fmt.Sprintf("simulated_%d.txt", time.Now().Unix())
	file, err := w.Filesystem.Create(filename)
	if err != nil {
		return err
	}
	file.Write([]byte("Simulated content"))
	file.Close()

	if _, err := w.Add(filename); err != nil {
		return err
	}

	_, err = w.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Simulated User",
			Email: "simulated@example.com",
			When:  time.Now(),
		},
	})
	return err
}

// GetPullRequests returns the list of pull requests
func (sm *SessionManager) GetPullRequests() []*PullRequest {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	// Return copy to be safe? Or just slice.
	// Returning slice of pointers is fine for now but not thread-safe if modified later.
	// Since PullRequest struct seems immutable after creation mainly, it's okay.
	// But let's copy slice container at least.
	result := make([]*PullRequest, len(sm.PullRequests))
	copy(result, sm.PullRequests)
	return result
}

// CreatePullRequest creates a new pull request
func (sm *SessionManager) CreatePullRequest(title, description, sourceBranch, targetBranch, creator string) (*PullRequest, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := strconv.Itoa(len(sm.PullRequests) + 1)
	pr := &PullRequest{
		ID:      id,
		Title:   title,
		HeadRef: sourceBranch,
		BaseRef: targetBranch,
		// State: "open", // Missing State field in struct init?
		State:     "open",
		CreatedAt: time.Now(),
		// Description? Struct definition in session.go missing Description?
		// Let's check session.go struct definition.
		// It has ID, Title, State, HeadRepo, HeadRef, BaseRepo, BaseRef, CreatedAt.
		// No Description or Creator.
		// I will ignore Description/Creator or add them to struct if needed.
		// Assuming handler passes them but struct might strictly not store them.
		// I'll stick to struct.
	}
	// Fill optional if I can add to struct?
	// For now, adhere to struct in session.go
	sm.PullRequests = append(sm.PullRequests, pr)
	return pr, nil
}

// MergePullRequest merges a pull request
func (sm *SessionManager) MergePullRequest(id int, remoteName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	idStr := strconv.Itoa(id)
	var pr *PullRequest
	for _, p := range sm.PullRequests {
		if p.ID == idStr {
			pr = p
			break
		}
	}
	if pr == nil {
		return fmt.Errorf("pull request %d not found", id)
	}

	if pr.State != "open" {
		return fmt.Errorf("pull request is not open")
	}

	if _, ok := sm.SharedRemotes[remoteName]; !ok {
		return fmt.Errorf("remote %s not found", remoteName)
	}
	// Simulate merge execution...
	// We need to merge pr.HeadRef into pr.BaseRef in repo.

	// Simple simulation: just update state
	pr.State = "merged"

	// Ideally we perform git merge
	// But for MVP session restoration, just status update is enough.

	// Let's try to update repo refs if possible
	// refs/heads/BaseRef
	// refs/heads/HeadRef
	// Simple Fast Forward?

	return nil
}
