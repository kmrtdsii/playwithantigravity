package commands

// clone.go - Simulated Git Clone Command
//
// IMPORTANT: This implementation does NOT clone from real network URLs.
// It looks up SharedRemotes (pre-ingested virtual remotes) or creates
// a simulated remote from the URL. Objects are copied in-memory.

import (
	"context"
	"fmt"
	"log"
	"os"
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
	// DEBUG: Log to file
	logFile := "/tmp/gitgym_clone_debug.log"
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Clone: Failed to open log file %s: %v", logFile, err)
	} else {
		defer f.Close()
		log.SetOutput(f)
		log.Printf("----------------------------------------------------------------")
		log.Printf("Clone: Logging initialized to %s", logFile)
	}
	log.Printf("Clone: Starting execution args=%v", args)

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
		log.Printf("Clone: Checking shared remotes for %s", url)

		if r, ok := s.Manager.GetSharedRemote(url); ok { // SAFE ACCESS
			log.Printf("Clone: Found shared remote for URL %s", url)
			remoteRepo = r
			remoteSt = r.Storer

			// Look up the internal path to ensure we point to the bare repo, NOT the external URL
			s.Manager.RLock()
			path, found := s.Manager.SharedRemotePaths[url]
			s.Manager.RUnlock()

			if found {
				remotePath = path
			} else {
				remotePath = url // Fallback, though this implies leakage risk if it's a real URL
			}
		} else if r, ok := s.Manager.GetSharedRemote(repoName); ok { // SAFE ACCESS
			log.Printf("Clone: Found shared remote by name %s", repoName)
			remoteRepo = r
			remoteSt = r.Storer

			s.Manager.RLock()
			path, found := s.Manager.SharedRemotePaths[repoName]
			s.Manager.RUnlock()

			if found {
				remotePath = path
			} else {
				remotePath = repoName
			}
		}
	}

	if remoteRepo == nil {
		// RESTRICTION: Disable arbitrary network cloning to prevent hangs.
		// Users must only use pre-configured SharedRemotes.
		return "", fmt.Errorf("repository '%s' not found in shared remotes. Network cloning is disabled to prevent timeout issues. Please use a valid shared remote URL.", url)
	}

	// 2. Create Local Working Copy
	log.Printf("Clone: Remote resolved. Path: %s. Starting Local Creation...", remotePath)

	if errMkdir := s.Filesystem.MkdirAll(repoName, 0755); errMkdir != nil {
		return "", fmt.Errorf("failed to create directory: %w", errMkdir)
	}

	repoFS, err := s.Filesystem.Chroot(repoName)
	if err != nil {
		return "", fmt.Errorf("failed to chroot: %w", err)
	}

	// PERFORMANCE: Use Filesystem Storage for Local Repo (.git)
	// Create the .git directory
	if errDotGit := repoFS.MkdirAll(".git", 0755); errDotGit != nil {
		return "", fmt.Errorf("failed to create .git directory: %w", errDotGit)
	}
	dotGitFS, err := repoFS.Chroot(".git")
	if err != nil {
		return "", fmt.Errorf("failed to chroot .git: %w", err)
	}

	localSt := filesystem.NewStorage(dotGitFS, cache.NewObjectLRUDefault())

	// OPTIMIZATION: Use HybridStorer to avoid copying objects
	// This delegates object reads to the remoteSt if not found locally.
	hybridSt := git.NewHybridStorer(localSt, remoteSt)

	localRepo, err := gogit.Init(hybridSt, repoFS)
	if err != nil {
		return "", fmt.Errorf("failed to init local repo: %w", err)
	}

	// Copy Objects STEP REMOVED - HybridStorer handles it dynamically
	log.Printf("Clone: Using HybridStorer (Zero-Copy). Local initialized.")

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
			_ = localRepo.Storer.SetReference(newRef)
		} else if name.IsTag() {
			// Copy tags as is
			_ = localRepo.Storer.SetReference(ref)
		}
		return nil
	})

	// Determine appropriate URL for origin
	// Use the original URL to prevent exposing internal paths in error messages
	originURL := url
	// If it's not a URL schema and not absolute, assume it's relative to root
	if !strings.Contains(originURL, "://") && !strings.HasPrefix(originURL, "/") {
		originURL = "/" + originURL
	}

	// Set Origin to point to our simulated remote path (mapped via SharedRemotes)
	// We use the friendly URL; Fetch/Push commands will resolve it.
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
				_ = localRepo.Storer.SetReference(newBranchRef)

				_ = w.Checkout(&gogit.CheckoutOptions{
					Branch: branchRef,
					Create: false, // Created manually above
					Force:  true,
				})
				break
			}
		}
	}

	log.Printf("Clone: Success. Cloned into %s", repoName)
	return fmt.Sprintf("Cloned into '%s'... (Using shared remote %s)", repoName, remotePath), nil
}

func (c *CloneCommand) Help() string {
	return `usage: git clone <url>

Clone a repository into a new directory.

Note: This is simulated cloning from virtual shared remotes.
No actual network operations are performed.
`
}
