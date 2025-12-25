package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(ctx context.Context, name, url string) error {
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

	// 1.5. Capture Old Paths for Pruning Stale Workspaces
	sm.mu.Lock()
	oldPaths := make(map[string]bool)
	for k, v := range sm.SharedRemotePaths {
		oldPaths[k] = true // Capture URL/Name (Keys)
		oldPaths[v] = true // Capture Resolved Path (Values) - just in case
	}
	sm.mu.Unlock()

	// 2. Clear InMemory Maps - Needs LOCK
	sm.mu.Lock()
	sm.SharedRemotes = make(map[string]*gogit.Repository)
	sm.SharedRemotePaths = make(map[string]string)
	sm.mu.Unlock() // Release lock before cloning

	// ensure target dir exists

	if errMkdir := os.MkdirAll(repoPath, 0755); errMkdir != nil {
		return fmt.Errorf("failed to create remote dir: %w", errMkdir)
	}

	// 3. Open or Clone
	// ALWAYS force a fresh clone for now to ensure we get a Mirror (all refs).
	// In the future, we could open and Fetch, but for "Ingest/Update" action, full sync is safer.
	os.RemoveAll(repoPath)

	log.Printf("IngestRemote: Cloning %s into %s", url, repoPath)
	repo, err := gogit.PlainCloneContext(ctx, repoPath, true, &gogit.CloneOptions{
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

	// Store under Internal Path (so fetches using internal path work)
	sm.SharedRemotes[repoPath] = repo
	sm.SharedRemotePaths[repoPath] = repoPath

	// 5. Prune Stale Workspaces
	// We do this AFTER adding the new one, but logic relies on oldPaths captured BEFORE clear.
	go sm.pruneStaleWorkspaces(oldPaths)

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

	id := sm.NextPRID
	sm.NextPRID++
	pr := &PullRequest{
		ID:          id,
		Title:       title,
		HeadRef:     sourceBranch,
		BaseRef:     targetBranch,
		State:       "OPEN",
		Description: description,
		Creator:     creator,
		CreatedAt:   time.Now(),
	}
	sm.PullRequests = append(sm.PullRequests, pr)
	return pr, nil
}

// DeletePullRequest removes a pull request by ID
func (sm *SessionManager) DeletePullRequest(id int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, pr := range sm.PullRequests {
		if pr.ID == id {
			// Delete preserving order not strictly required, but usually good.
			// Fast delete:
			// sm.PullRequests[i] = sm.PullRequests[len(sm.PullRequests)-1]
			// sm.PullRequests = sm.PullRequests[:len(sm.PullRequests)-1]
			//
			// Preserving order (better for UI stability):
			sm.PullRequests = append(sm.PullRequests[:i], sm.PullRequests[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pull request %d not found", id)
}

// pruneStaleWorkspaces removes local repos that point to deleted shared remotes
func (sm *SessionManager) pruneStaleWorkspaces(stalePaths map[string]bool) {
	if len(stalePaths) == 0 {
		return
	}

	sm.mu.RLock()
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.mu.RUnlock()

	for _, s := range sessions {
		s.Lock()
		var reposToRemove []string

		for name, repo := range s.Repos {
			remotes, err := repo.Remotes()
			if err != nil {
				continue
			}
			for _, r := range remotes {
				for _, url := range r.Config().URLs {
					if stalePaths[url] {
						reposToRemove = append(reposToRemove, name)
						break
					}
				}
			}
		}

		for _, name := range reposToRemove {
			log.Printf("Pruning stale workspace: %s (Session %s)", name, s.ID)
			delete(s.Repos, name)
			// Remove from filesystem
			_ = s.RemoveAll(name)

			// Reset CurrentDir if inside deleted repo
			// Check if CurrentDir starts with /name
			if s.CurrentDir == "/"+name || (len(s.CurrentDir) > len(name)+2 && s.CurrentDir[:len(name)+2] == "/"+name+"/") {
				s.CurrentDir = "/"
			}
		}
		s.Unlock()
	}
}
