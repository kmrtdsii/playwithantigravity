package commands

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("diff", func() git.Command { return &DiffCommand{} })
}

type DiffCommand struct{}

func (c *DiffCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	if len(args) < 3 {
		return "usage: git diff <ref1> <ref2>\n(Worktree diff not yet supported)", nil
	}
	ref1 := args[1]
	ref2 := args[2]

	// Resolve refs
	h1, err := repo.ResolveRevision(plumbing.Revision(ref1))
	if err != nil {
		return "", err
	}
	h2, err := repo.ResolveRevision(plumbing.Revision(ref2))
	if err != nil {
		return "", err
	}

	c1, err := repo.CommitObject(*h1)
	if err != nil {
		return "", err
	}
	c2, err := repo.CommitObject(*h2)
	if err != nil {
		return "", err
	}

	tree1, err := c1.Tree()
	if err != nil {
		return "", err
	}
	tree2, err := c2.Tree()
	if err != nil {
		return "", err
	}

	patch, err := tree1.Patch(tree2)
	if err != nil {
		return "", err
	}

	return patch.String(), nil
}

func (c *DiffCommand) Help() string {
	return "usage: git diff <ref1> <ref2>"
}
