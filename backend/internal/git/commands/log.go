package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("log", func() git.Command { return &LogCommand{} })
}

type LogCommand struct{}

func (c *LogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	oneline := false
	if len(args) > 1 && args[1] == "--oneline" {
		oneline = true
	}

	cIter, err := repo.Log(&gogit.LogOptions{All: false}) // HEAD only usually
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = cIter.ForEach(func(c *object.Commit) error {
		if oneline {
			// 7-char hash + message
			sb.WriteString(fmt.Sprintf("%s %s\n", c.Hash.String()[:7], strings.Split(c.Message, "\n")[0]))
		} else {
			sb.WriteString(fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n\n",
				c.Hash.String(),
				c.Author.Name,
				c.Author.Email,
				c.Author.When.Format(time.RFC3339),
				strings.TrimSpace(c.Message),
			))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (c *LogCommand) Help() string {
	return "usage: git log [--oneline]\n\nShow commit logs."
}
