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
	"github.com/go-git/go-git/v5/config"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(ctx context.Context, name, url string, depth int) error {
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

	// Serialized Ingestion to prevent race conditions (main vs frontend)
	sm.ingestMu.Lock()
	defer sm.ingestMu.Unlock()

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
	// Do NOT clear maps yet. We only update them on success.
	sm.mu.Unlock()

	// 2. Check if already exists and is valid
	var repo *gogit.Repository
	if _, errStat := os.Stat(repoPath); errStat == nil {
		// Try opening
		r, errOpen := gogit.PlainOpen(repoPath)
		if errOpen == nil {
			log.Printf("IngestRemote: Repository already exists at %s. Fetching updates...", repoPath)

			// FIX: Ensure we are NOT in Mirror mode (which fetches all PR refs).
			// If previously initialized with Mirror: true, the config will have fetch = +refs/*:refs/*.
			// We must reset it to only fetch heads and tags.
			cfg, errCfg := r.Config()
			if errCfg == nil && cfg.Remotes["origin"] != nil {
				// Force sensible refspecs
				// For a "bare remote" simulation, we want to mirror branches and tags, but NOT everything (PRs).
				// +refs/heads/*:refs/heads/* and +refs/tags/*:refs/tags/*
				refsHeads := config.RefSpec("+refs/heads/*:refs/heads/*")
				refsTags := config.RefSpec("+refs/tags/*:refs/tags/*")

				needsUpdate := false
				if cfg.Remotes["origin"].Mirror {
					cfg.Remotes["origin"].Mirror = false
					needsUpdate = true
				}
				// Replace Fetch specs if they look like the dangerous wildcard
				if len(cfg.Remotes["origin"].Fetch) == 0 || cfg.Remotes["origin"].Fetch[0].String() == "+refs/*:refs/*" {
					cfg.Remotes["origin"].Fetch = []config.RefSpec{refsHeads, refsTags}
					needsUpdate = true
				}

				if needsUpdate {
					if errSet := r.SetConfig(cfg); errSet != nil {
						log.Printf("IngestRemote: Failed to update config to disable mirror: %v", errSet)
					} else {
						log.Printf("IngestRemote: Updated remote config to disable Mirror mode (PR refs)")
					}
				}
			}

			// It exists. Fetch to update refs.
			errFetch := r.Fetch(&gogit.FetchOptions{
				Progress: os.Stdout,
				Force:    true, // Force update refs
				Tags:     gogit.AllTags,
			})
			if errFetch != nil && errFetch != gogit.NoErrAlreadyUpToDate {
				log.Printf("IngestRemote: Fetch failed (%v), falling back to fresh clone", errFetch)
				// Fallthrough to clone is risky if we have bad config, but we just fixed config.
				// If fetch failed, maybe the repo is corrupt. Let's recreate.
				repo = nil // Signal to re-clone
			} else {
				log.Printf("IngestRemote: Fetch successful or already up to date")
				repo = r
			}
		}
	}

	// 3. Clone if not opened successfully
	if repo == nil {
		// Clear directory to be safe
		os.RemoveAll(repoPath)
		if errMkdir := os.MkdirAll(repoPath, 0755); errMkdir != nil {
			return fmt.Errorf("failed to create remote dir: %w", errMkdir)
		}

		log.Printf("IngestRemote: Cloning %s into %s (Depth: %d)", url, repoPath, depth)

		// Setup clone options
		// IMPORTANT: Do NOT use Mirror: true. It fetches +refs/*:refs/* which includes thousands of PRs for popular repos.
		cloneOpts := &gogit.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
			// Mirror:   true, // DANGEROUS for shared repos
			Depth: depth,
			Tags:  gogit.AllTags,
		}

		r, errClone := gogit.PlainCloneContext(ctx, repoPath, true, cloneOpts)
		if errClone != nil {
			return fmt.Errorf("failed to clone remote: %w", errClone)
		}

		// Post-clone: Configure fetch refspecs explicitly for future consistency?
		// Default PlainClone(bare=true) usually sets +refs/heads/*:refs/heads/*.
		// We verify this is sufficient.

		repo = r
		log.Printf("IngestRemote: Clone successful")
	}

	// 4. Update State - Needs LOCK
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Reset maps to ensure cleanup of old remotes in memory
	sm.SharedRemotes = make(map[string]*gogit.Repository)
	sm.SharedRemotePaths = make(map[string]string)

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
