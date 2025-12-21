package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
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
	// Simple parsing: last part of path, strip .git
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid url")
	}
	repoName := parts[len(parts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	if _, exists := s.Repos[repoName]; exists {
		return "", fmt.Errorf("repository '%s' already exists", repoName)
	}

	// Create chrooted filesystem for the repo
	if err := s.Filesystem.MkdirAll(repoName, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	repoFS, err := s.Filesystem.Chroot(repoName)
	if err != nil {
		return "", fmt.Errorf("failed to chroot: %w", err)
	}

	st := memory.NewStorage()
	repo, err := gogit.Clone(st, repoFS, &gogit.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		// Clean up on failure?
		return "", fmt.Errorf("clone failed: %w", err)
	}

	s.Repos[repoName] = repo

	return fmt.Sprintf("Cloned into '%s'...", repoName), nil
}

func (c *CloneCommand) Help() string {
	return "usage: git clone <url>\n\nClone a repository into a new directory." // We actually clone into root of filesystem for this session
}
