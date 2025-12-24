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
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("clone", func() git.Command { return &CloneCommand{} })
}

type CloneCommand struct{}

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
			} else {
				// clone only takes one arg (url) and optional dir?
				// "git clone <url> <dir>" support?
				// Legacy only supported <url>.
				// If we want to be strict, we can error. Or ignore.
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

	if _, exists := s.Repos[repoName]; exists {
		return "", fmt.Errorf("repository '%s' already exists", repoName)
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
		remotePath = fmt.Sprintf("remotes/%s.git", repoName)
		if err := s.Filesystem.MkdirAll("remotes", 0755); err != nil {
			return "", fmt.Errorf("failed to create remotes directory: %w", err)
		}

		remoteSt = memory.NewStorage()
		var err error
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

	localSt := memory.NewStorage()
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

	// Copy References
	refs, _ := remoteRepo.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		localRepo.Storer.SetReference(ref)
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
			if _, err := localRepo.Reference(branchRef, true); err == nil {
				w.Checkout(&gogit.CheckoutOptions{
					Branch: branchRef,
					Create: false,
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
