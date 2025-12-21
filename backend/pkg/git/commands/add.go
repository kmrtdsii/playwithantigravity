package commands

import (
	"context"
	"fmt"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("add", func() git.Command { return &AddCommand{} })
}

type AddCommand struct{}

func (c *AddCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := s.Repo.Worktree()
	if len(args) < 2 {
		return "", fmt.Errorf("usage: git add <file>")
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
	return "usage: git add <file>...\n\nAdd file contents to the index."
}
