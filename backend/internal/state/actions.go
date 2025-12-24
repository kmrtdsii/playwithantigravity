package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(name, url string) error {
	// Define local path for persistence
	// READ LOCK only to get config if needed, but DataDir is static usually or we can just access it if it's not changing.
	// Safe to read DataDir if it's set on init.
	baseDir := sm.DataDir
	if baseDir == "" {
		baseDir = ".gitgym-data/remotes"
	}

	// Requirement: Single persistent remote. Clean up others.
	// We use URL hash for directory name so clients pointing to old URL paths fail.
	hash := sha256.Sum256([]byte(url))
	dirName := hex.EncodeToString(hash[:])
	repoPath := filepath.Join(baseDir, dirName)
	if absPath, err := filepath.Abs(repoPath); err == nil {
		repoPath = absPath
	}

	// 1. Enforce Single Residency: Delete everything in baseDir that isn't our target
	// Create baseDir if not exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	entries, err := os.ReadDir(baseDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != dirName {
				// Remove old remote
				os.RemoveAll(filepath.Join(baseDir, entry.Name()))
			}
		}
	}

	// 2. Clear InMemory Maps - Needs LOCK
	sm.mu.Lock()
	sm.SharedRemotes = make(map[string]*gogit.Repository)
	sm.SharedRemotePaths = make(map[string]string)
	sm.mu.Unlock() // Release lock before cloning

	// ensure target dir exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return fmt.Errorf("failed to create remote dir: %w", err)
	}

	// 3. Open or Clone
	// ALWAYS force a fresh clone for now to ensure we get a Mirror (all refs).
	// In the future, we could open and Fetch, but for "Ingest/Update" action, full sync is safer.
	os.RemoveAll(repoPath)

	log.Printf("IngestRemote: Cloning %s into %s", url, repoPath)
	repo, err := gogit.PlainClone(repoPath, true, &gogit.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
		Mirror:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to clone remote: %w", err)
	}
	log.Printf("IngestRemote: Clone successful")

	// 4. Update State - Needs LOCK
	sm.mu.Lock()
	defer sm.mu.Unlock()

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
func (sm *SessionManager) SimulateCommit(remoteName, message, authorName, authorEmail string) error {
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

	if authorName == "" {
		authorName = "Simulated User"
	}
	if authorEmail == "" {
		authorEmail = "simulated@example.com"
	}

	_, err = w.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
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
