package commands

// add.go - Simulated Git Add Command
//
// Stages file contents to the index for the next commit.
// This operates on the in-memory worktree and index.

import (
	"context"
	"fmt"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("add", func() git.Command { return &AddCommand{} })
}

type AddCommand struct{}

func (c *AddCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Parse flags
	for _, arg := range args[1:] {
		if arg == "-h" || arg == "--help" {
			return c.Help(), nil
		}
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := repo.Worktree()
	if len(args) < 2 {
		return "", fmt.Errorf("Nothing specified, nothing added.\nMaybe you wanted to say 'git add .'?")
	}

	// args[0] is "add"
	file := args[1]
	var err error
	if file == "." {
		_, err = w.Add(".")
	} else {
		_, err = w.Add(file)
	}
	if err != nil {
		return "", err
	}
	return "Added " + file, nil
}

func (c *AddCommand) Help() string {
	return `usage: git add [options] [--] <pathspec>...

Options:
    .                 add all changes in current directory
    <file>            add specific file

Add file contents to the index (staging area).
`
}
