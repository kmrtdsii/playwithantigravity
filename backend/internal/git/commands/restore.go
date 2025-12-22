package commands

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("restore", func() git.Command { return &RestoreCommand{} })
}

type RestoreCommand struct{}

func (c *RestoreCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Just use system git for 'restore' command as it handles pathspecs and modes robustly
	// args includes the command "restore" as the first element usually.
	return runnerExecute(ctx, s.CurrentDir, args)
}

func runnerExecute(ctx context.Context, dir string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git restore failed: %v\nOutput: %s", err, string(out))
	}
	return string(out), nil
}

func (c *RestoreCommand) Help() string {
	return "usage: git restore [--staged] <file>"
}
