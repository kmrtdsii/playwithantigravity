package state

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(name, url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// In a real scenario, we would Clone from the URL.
	// For simulation, we'll creates an empty repo or clone if the URL works?
	// Given network restrictions or simulation goals, let's try to clone if it's a real URL,
	// otherwise just init an empty one with that name.

	// Use memory storage
	storage := memory.NewStorage()
	fs := memfs.New()

	// Try cloning
	repo, err := gogit.Clone(storage, fs, &gogit.CloneOptions{
		URL: url,
	})
	if err != nil {
		// Fallback for simulation: Init empty
		repo, err = gogit.Init(storage, fs)
		if err != nil {
			return err
		}
	}

	sm.SharedRemotes[name] = repo
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
