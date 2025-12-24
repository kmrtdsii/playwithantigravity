package commands

// clone.go - Simulated Git Clone Command
//
// IMPORTANT: This implementation does NOT clone from real network URLs.
// It looks up SharedRemotes (pre-ingested virtual remotes) or creates
// a simulated remote from the URL. Objects are copied in-memory.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("clone", func() git.Command { return &CloneCommand{} })
}

type CloneCommand struct{}

// SafeRepoNameRegex enforces alphanumeric names to prevent traversal
var SafeRepoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func (c *CloneCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Parse flags
	var url string

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-h", "--help":
			return c.Help(), nil
		default:
			if url == "" {
				url = arg
			}
		}
	}

	if url == "" {
		return "", fmt.Errorf("usage: git clone <url>")
	}

	// Extract repo name from URL
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid url")
	}
	repoName := parts[len(parts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	// SECURITY: Input Validation
	if !SafeRepoNameRegex.MatchString(repoName) {
		return "", fmt.Errorf("invalid repository name '%s': must contain only alphanumeric characters, underscores, or hyphens", repoName)
	}
	if repoName == "." || repoName == ".." {
		return "", fmt.Errorf("invalid repository name: cannot be relative path")
	}

	if _, exists := s.Repos[repoName]; exists {
		return "", fmt.Errorf("destination path '%s' already exists and is not an empty directory", repoName)
	}

	// 1. Resolve Remote Repository (Shared vs Session-local)
	var remoteRepo *gogit.Repository
	var remoteSt storage.Storer
	var remotePath string

	if s.Manager != nil {
		// Check SharedRemotes (e.g., "origin", or the full URL)
		if r, ok := s.Manager.SharedRemotes[url]; ok {
			remoteRepo = r
			remoteSt = r.Storer
			// Check if we have a physical path for this remote
			if path, hasPath := s.Manager.SharedRemotePaths[url]; hasPath {
				remotePath = path
			} else {
				remotePath = url
			}
		} else if r, ok := s.Manager.SharedRemotes[repoName]; ok {
			remoteRepo = r
			remoteSt = r.Storer
			// Check if we have a physical path for this remote
			if path, hasPath := s.Manager.SharedRemotePaths[repoName]; hasPath {
				remotePath = path
			} else {
				remotePath = repoName
			}
		}
	}

	if remoteRepo == nil {
		// Fallback: Create Simulated Remote (Legacy behavior)
		// SECURITY: Prevent traversal in path construction
		repoNameClean := filepath.Clean(repoName)
		if strings.Contains(repoNameClean, "..") {
			return "", fmt.Errorf("security violation: invalid remote path")
		}

		remotePath = fmt.Sprintf("remotes/%s.git", repoNameClean)
		if err := s.Filesystem.MkdirAll("remotes", 0755); err != nil {
			return "", fmt.Errorf("failed to create remotes directory: %w", err)
		}

		// PERFORMANCE: Use Filesystem Storage for Remote
		remoteDot, err := s.Filesystem.Chroot(remotePath)
		if err != nil {
			return "", fmt.Errorf("failed to chroot for remote: %w", err)
		}

		remoteSt = filesystem.NewStorage(remoteDot, cache.NewObjectLRUDefault())

		remoteRepo, err = gogit.Clone(remoteSt, nil, &gogit.CloneOptions{
			URL:      url,
			Progress: os.Stdout,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create simulated remote: %w", err)
		}
		s.Repos[remotePath] = remoteRepo
	}

	// 2. Create Local Working Copy
	if err := s.Filesystem.MkdirAll(repoName, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	repoFS, err := s.Filesystem.Chroot(repoName)
	if err != nil {
		return "", fmt.Errorf("failed to chroot: %w", err)
	}

	// PERFORMANCE: Use Filesystem Storage for Local Repo (.git)
	// Create the .git directory
	if err := repoFS.MkdirAll(".git", 0755); err != nil {
		return "", fmt.Errorf("failed to create .git directory: %w", err)
	}
	dotGitFS, err := repoFS.Chroot(".git")
	if err != nil {
		return "", fmt.Errorf("failed to chroot .git: %w", err)
	}

	localSt := filesystem.NewStorage(dotGitFS, cache.NewObjectLRUDefault())
	localRepo, err := gogit.Init(localSt, repoFS)
	if err != nil {
		return "", fmt.Errorf("failed to init local repo: %w", err)
	}

	// Copy Objects from remote to local
	iter, _ := remoteSt.IterEncodedObjects(plumbing.AnyObject)
	iter.ForEach(func(obj plumbing.EncodedObject) error {
		localSt.SetEncodedObject(obj)
		return nil
	})

	// Copy References with Mapping (Standard Git Behavior)
	// refs/heads/* -> refs/remotes/origin/*
	// refs/tags/*  -> refs/tags/*
	refs, _ := remoteRepo.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		if name.IsBranch() {
			// Map refs/heads/foo -> refs/remotes/origin/foo
			newRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", name.Short()))
			newRef := plumbing.NewHashReference(newRefName, ref.Hash())
			localRepo.Storer.SetReference(newRef)
		} else if name.IsTag() {
			// Copy tags as is
			localRepo.Storer.SetReference(ref)
		}
		return nil
	})

	// Determine appropriate URL for origin
	originURL := remotePath
	// If it's not a URL schema and not absolute, assume it's relative to root
	if !strings.Contains(originURL, "://") && !strings.HasPrefix(originURL, "/") {
		originURL = "/" + originURL
	}

	// Set Origin to point to our simulated remote path
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{originURL},
	})
	if err != nil {
		return "", fmt.Errorf("failed to configure origin: %w", err)
	}

	s.Repos[repoName] = localRepo

	// 3. Auto-cd into the cloned repository
	s.CurrentDir = "/" + repoName

	// 4. Checkout the default branch (main or master)
	w, err := localRepo.Worktree()
	if err == nil {
		// Try main first, then master
		for _, branch := range []string{"main", "master"} {
			branchRef := plumbing.NewBranchReferenceName(branch)
			// Check if we have the remote counterpart: refs/remotes/origin/branch
			remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", branch))
			if _, err := localRepo.Reference(remoteRefName, true); err == nil {
				// Create local branch 'branch' pointing to the same hash
				// And checkout
				ref, _ := localRepo.Reference(remoteRefName, true)
				newBranchRef := plumbing.NewHashReference(branchRef, ref.Hash())
				localRepo.Storer.SetReference(newBranchRef)

				w.Checkout(&gogit.CheckoutOptions{
					Branch: branchRef,
					Create: false, // Created manually above
					Force:  true,
				})
				break
			}
		}
	}

	return fmt.Sprintf("Cloned into '%s'... (Using shared remote %s)", repoName, remotePath), nil
}

func (c *CloneCommand) Help() string {
	return `usage: git clone <url>

Clone a repository into a new directory.

Note: This is simulated cloning from virtual shared remotes.
No actual network operations are performed.
`
}
