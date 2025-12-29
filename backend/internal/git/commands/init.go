package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("init", func() git.Command { return &InitCommand{} })
}

type InitCommand struct{}

// Ensure InitCommand implements git.Command
var _ git.Command = (*InitCommand)(nil)

func (c *InitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Determine path
	var rawPath string
	if len(args) > 1 {
		rawPath = args[1]
	}

	var path string // Resolved path without leading slash

	var targetPath string
	if rawPath == "" {
		targetPath = s.CurrentDir
	} else {
		if strings.HasPrefix(rawPath, "/") {
			targetPath = rawPath
		} else {
			// Join handles cleaning and expected separators for logical paths
			if s.CurrentDir == "/" {
				targetPath = "/" + rawPath
			} else {
				targetPath = s.CurrentDir + "/" + rawPath
			}
		}
	}

	if targetPath == "/" {
		return "", fmt.Errorf("cannot init repository at root. Run 'mkdir <name>' first, then 'cd <name>' and 'git init'")
	}

	// Remove leading slash for internal handling
	path = strings.TrimPrefix(targetPath, "/")

	// Check for nested repository conflicts
	if err := c.checkNestedRepoConflicts(s, path); err != nil {
		return "", err
	}

	_, err := s.InitRepo(path)
	if err != nil {
		return "", fmt.Errorf("failed to init repo: %w", err)
	}

	return fmt.Sprintf("Initialized empty Git repository in /%s/.git/", path), nil
}

// checkNestedRepoConflicts checks if the target path would create a nested repository
func (c *InitCommand) checkNestedRepoConflicts(s *git.Session, targetPath string) error {
	// Normalize target path (ensure no leading slash for comparison)
	targetPath = strings.TrimPrefix(targetPath, "/")

	for existingPath := range s.Repos {
		existingPath = strings.TrimPrefix(existingPath, "/")

		// Check if target is inside an existing repo (parent repo exists)
		if strings.HasPrefix(targetPath, existingPath+"/") {
			return fmt.Errorf("cannot init repository inside existing repo '/%s'", existingPath)
		}

		// Check if existing repo is inside target (child repo would be nested)
		if strings.HasPrefix(existingPath, targetPath+"/") {
			return fmt.Errorf("cannot init repository: nested repo exists at '/%s'", existingPath)
		}

		// Check if same path (reinitializing)
		if targetPath == existingPath {
			return fmt.Errorf("repository already exists at '/%s'", existingPath)
		}
	}

	return nil
}

func (c *InitCommand) Help() string {
	return "usage: git init [directory]\n\nCreate an empty Git repository or reinitialize an existing one."
}
