package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("reflog", func() git.Command { return &ReflogCommand{} })
}

type ReflogCommand struct{}

func (c *ReflogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	var sb strings.Builder
	for i, entry := range s.Reflog {
		sb.WriteString(fmt.Sprintf("%s HEAD@{%d}: %s\n", entry.Hash[:7], i, entry.Message))
	}
	return sb.String(), nil
}

func (c *ReflogCommand) Help() string {
	return "usage: git reflog\n\nShow reflog entries."
}
