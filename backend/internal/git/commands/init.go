package commands

import (
	"context"
	"fmt"
	"path"
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

	// Parse optional path argument
	var argPath string
	if len(args) > 1 {
		argPath = args[1]
	}

	// Resolve target path (always absolute, starting with /)
	var targetPath string
	if argPath == "" {
		// No argument: init in current directory
		targetPath = s.CurrentDir
	} else if strings.HasPrefix(argPath, "/") {
		// Absolute path provided
		targetPath = path.Clean(argPath)
	} else {
		// Relative path: join with current directory
		targetPath = path.Clean(path.Join(s.CurrentDir, argPath))
	}

	// Validate: cannot init at root
	if targetPath == "/" {
		return "", fmt.Errorf("cannot init repository at root. Run 'mkdir <name>' first, then 'cd <name>' and 'git init'")
	}

	// Convert to internal path format (without leading slash)
	internalPath := strings.TrimPrefix(targetPath, "/")

	// Check for nested repository conflicts
	if err := c.checkNestedRepoConflicts(s, internalPath); err != nil {
		return "", err
	}

	_, err := s.InitRepo(internalPath)
	if err != nil {
		return "", fmt.Errorf("failed to init repo: %w", err)
	}

	return fmt.Sprintf("Initialized empty Git repository in /%s/.git/", internalPath), nil
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
