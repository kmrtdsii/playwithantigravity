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
	"github.com/go-git/go-git/v5/plumbing"
)

// IngestRemote creates a new shared remote repository from a URL (simulated clone)
func (sm *SessionManager) IngestRemote(ctx context.Context, name, url string, depth int) error {
	// Define local path for persistence
	baseDir := os.Getenv("GITGYM_DATA_ROOT")
	if baseDir == "" {
		baseDir = ".gitgym-data"
	}
	baseDir = filepath.Join(baseDir, "remotes")

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

	// 1. Ensure Base Directory exists
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	// 1.5. Capture Old Paths for Pruning Stale Workspaces - DISABLED
	// sm.mu.Lock()
	// oldPaths := make(map[string]bool)
	// for k, v := range sm.SharedRemotePaths {
	// 	oldPaths[k] = true // Capture URL/Name (Keys)
	// 	oldPaths[v] = true // Capture Resolved Path (Values) - just in case
	// }
	// // Do NOT clear maps yet. We only update them on success.
	// sm.mu.Unlock()

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
				refsHeads := config.RefSpec("+refs/heads/*:refs/heads/*")
				refsTags := config.RefSpec("+refs/tags/*:refs/tags/*")
				needsUpdate := false

				if cfg.Remotes["origin"].Mirror {
					cfg.Remotes["origin"].Mirror = false
					needsUpdate = true
				}

				// Aggressively ensure we have the right refspecs for a simulated server.
				// We want remote heads to be local heads in this bare repo so PRs work.
				isCorrect := len(cfg.Remotes["origin"].Fetch) == 2 &&
					cfg.Remotes["origin"].Fetch[0] == refsHeads &&
					cfg.Remotes["origin"].Fetch[1] == refsTags

				if !isCorrect {
					cfg.Remotes["origin"].Fetch = []config.RefSpec{refsHeads, refsTags}
					needsUpdate = true
				}

				if needsUpdate {
					if errSet := r.SetConfig(cfg); errSet != nil {
						log.Printf("IngestRemote: Failed to update config: %v", errSet)
					} else {
						log.Printf("IngestRemote: Updated remote config for bare server simulation")
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

				// Cleanup stale refs/remotes/* entries that might exist from previous mirror clones.
				// These cause duplicate labels ("main" and "origin/main") on the same commit.
				refs, refErr := r.References()
				if refErr == nil {
					var staleRefs []plumbing.ReferenceName
					_ = refs.ForEach(func(ref *plumbing.Reference) error {
						if ref.Name().IsRemote() {
							staleRefs = append(staleRefs, ref.Name())
						}
						return nil
					})
					for _, refName := range staleRefs {
						_ = r.Storer.RemoveReference(refName)
					}
					if len(staleRefs) > 0 {
						log.Printf("IngestRemote: Cleaned up %d stale remote refs", len(staleRefs))
					}
				}

				repo = r
			}
		}
	}

	// 3. Clone if not opened successfully
	if repo == nil {
		// Clear directory to be safe
		_ = os.RemoveAll(repoPath)
		if errMkdir := os.MkdirAll(repoPath, 0750); errMkdir != nil {
			return fmt.Errorf("failed to create remote dir: %w", errMkdir)
		}

		log.Printf("IngestRemote: Cloning %s into %s (Depth: %d)", url, repoPath, depth)

		// Setup clone options
		cloneOpts := &gogit.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
			Depth:    depth,
			Tags:     gogit.AllTags,
		}

		r, errClone := gogit.PlainCloneContext(ctx, repoPath, true, cloneOpts)
		if errClone != nil {
			return fmt.Errorf("failed to clone remote: %w", errClone)
		}

		// Post-clone: Fix refspecs to map remote heads to local heads (bare repo behavior)
		cfg, errCfg := r.Config()
		if errCfg == nil && cfg.Remotes["origin"] != nil {
			cfg.Remotes["origin"].Fetch = []config.RefSpec{
				"+refs/heads/*:refs/heads/*",
				"+refs/tags/*:refs/tags/*",
			}
			cfg.Remotes["origin"].Mirror = false
			if errSet := r.SetConfig(cfg); errSet != nil {
				log.Printf("IngestRemote: Failed to update config post-clone: %v", errSet)
			}
		}

		// Force fetch with new refspecs
		errFetch := r.Fetch(&gogit.FetchOptions{
			Force: true,
			Tags:  gogit.AllTags,
		})
		if errFetch != nil && errFetch != gogit.NoErrAlreadyUpToDate {
			log.Printf("IngestRemote: Post-clone fetch failed: %v", errFetch)
		}

		repo = r
		log.Printf("IngestRemote: Clone and refspec fix successful")
	}

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

	// 5. Prune Stale Workspaces - DISABLED
	// go sm.pruneStaleWorkspaces(oldPaths)

	return nil
}

// RemoveRemote removes a shared remote and cleans up all shared remotes (Single Residency)
func (sm *SessionManager) RemoveRemote(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 1. Resolve Path and Clean up disk if it exists
	path, ok := sm.SharedRemotePaths[name]
	if ok && path != "" {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("RemoveRemote: Failed to delete path %s: %v", path, err)
		} else {
			log.Printf("RemoveRemote: Deleted path %s", path)
		}
	}

	// 2. Clear specific entries in SharedRemotes
	delete(sm.SharedRemotes, name)
	delete(sm.SharedRemotePaths, name)

	// Clean up related mappings (URL, Path aliases)
	for k, v := range sm.SharedRemotePaths {
		if v == path {
			delete(sm.SharedRemotes, k)
			delete(sm.SharedRemotePaths, k)
		}
	}

	// 3. Clear associated pull requests
	var keptPRs []*PullRequest
	for _, pr := range sm.PullRequests {
		if pr.RemoteName != name {
			keptPRs = append(keptPRs, pr)
		}
	}
	sm.PullRequests = keptPRs

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
func (sm *SessionManager) CreatePullRequest(title, description, sourceBranch, targetBranch, creator, remoteName string) (*PullRequest, error) {
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
		RemoteName:  remoteName,
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

// CreateBareRepository creates a new bare repository on the server
// This only creates the remote repository - users must manually git clone or git init
func (sm *SessionManager) CreateBareRepository(ctx context.Context, sessionID, name string) error {
	// 1. Validate Name (Simple alphanumeric check)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("invalid repository name: only alphanumeric, hyphen and underscore allowed")
		}
	}

	// Define local path for persistence
	baseDir := os.Getenv("GITGYM_DATA_ROOT")
	if baseDir == "" {
		baseDir = ".gitgym-data"
	}
	baseDir = filepath.Join(baseDir, "remotes")

	// Use name or hash for directory? "remote://gitgym/{name}" -> hash
	// To be consistent with IngestRemote which hashes the URL.
	// Let's construct a pseudo-URL for consistency.
	pseudoURL := fmt.Sprintf("remote://gitgym/%s.git", name)
	hash := sha256.Sum256([]byte(pseudoURL))
	dirName := hex.EncodeToString(hash[:])
	repoPath := filepath.Join(baseDir, dirName)
	if absPath, err := filepath.Abs(repoPath); err == nil {
		repoPath = absPath
	}

	sm.ingestMu.Lock()
	defer sm.ingestMu.Unlock()

	// 2. Ensure Base Directory exists
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}

	// 3. Init Bare Repository
	// Always recreate to ensure empty state for "Create" action
	_ = os.RemoveAll(repoPath)
	if err := os.MkdirAll(repoPath, 0750); err != nil {
		return fmt.Errorf("failed to create repo dir: %w", err)
	}

	repo, err := gogit.PlainInit(repoPath, true) // isBare = true
	if err != nil {
		return fmt.Errorf("failed to init bare repo: %w", err)
	}

	// 4. Update Session Manager State
	sm.mu.Lock()

	// Register under Name, PseudoURL, and Path
	sm.SharedRemotes[name] = repo
	sm.SharedRemotePaths[name] = repoPath

	sm.SharedRemotes[pseudoURL] = repo
	sm.SharedRemotePaths[pseudoURL] = repoPath

	sm.SharedRemotes[repoPath] = repo
	sm.SharedRemotePaths[repoPath] = repoPath
	sm.mu.Unlock()

	log.Printf("Created bare repository: %s at %s", name, repoPath)

	return nil
}
