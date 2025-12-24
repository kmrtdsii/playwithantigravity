package commands

import (
	"context"
	"fmt"
	"runtime"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("version", func() git.Command { return &VersionCommand{} })
}

type VersionCommand struct{}

func (c *VersionCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// Imitate git version output
	// Example: git version 2.51.2
	return fmt.Sprintf("git version 2.51.2 (%s)", runtime.GOOS), nil
}

func (c *VersionCommand) Help() string {
	return "usage: git --version\n\nPrint the git version."
}
