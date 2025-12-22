package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("reflog", func() git.Command { return &ReflogCommand{} })
}

type ReflogCommand struct{}

func (c *ReflogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
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
