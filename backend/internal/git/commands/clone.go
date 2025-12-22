package commands

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
	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("clone", func() git.Command { return &CloneCommand{} })
}

type CloneCommand struct{}

func (c *CloneCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git clone <url>")
	}

	// Ensure we are in root
	if s.CurrentDir != "/" && s.CurrentDir != "" {
		return "", fmt.Errorf("git clone invalid permissions: you can only clone from the root directory")
	}

	url := args[1]

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
			remotePath = url
		} else if r, ok := s.Manager.SharedRemotes[repoName]; ok {
			remoteRepo = r
			remoteSt = r.Storer
			remotePath = repoName
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

	// Set Origin to point to our simulated remote path
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"/" + remotePath},
	})
	if err != nil {
		return "", fmt.Errorf("failed to configure origin: %w", err)
	}

	s.Repos[repoName] = localRepo

	return fmt.Sprintf("Cloned into '%s'... (Using shared remote %s)", repoName, remotePath), nil
}

func (c *CloneCommand) Help() string {
	return "usage: git clone <url>\n\nClone a repository into a new directory." // We actually clone into root of filesystem for this session
}
