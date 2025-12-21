package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("branch", func() git.Command { return &BranchCommand{} })
}

type BranchCommand struct{}

func (c *BranchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	if len(args) == 1 {
		// List branches
		iter, err := s.Repo.Branches()
		if err != nil {
			return "", err
		}
		var branches []string
		iter.ForEach(func(r *plumbing.Reference) error {
			branches = append(branches, r.Name().Short())
			return nil
		})
		return strings.Join(branches, "\n"), nil
	}

	// Handle branch deletion
	if args[1] == "-d" {
		if len(args) < 3 {
			return "", fmt.Errorf("branch name required")
		}
		branchName := args[2]

		// Validate branch exists
		refName := plumbing.ReferenceName("refs/heads/" + branchName)
		_, err := s.Repo.Reference(refName, true)
		if err != nil {
			return "", fmt.Errorf("branch '%s' not found.", branchName)
		}

		// Prevent deleting current branch
		headRef, err := s.Repo.Head()
		if err == nil && headRef.Name() == refName {
			return "", fmt.Errorf("cannot delete branch '%s' checked out at '%s'", branchName, "." /* worktree path info unavailable here */)
		}

		// Delete reference
		if err := s.Repo.Storer.RemoveReference(refName); err != nil {
			return "", err
		}
		return "Deleted branch " + branchName, nil
	}

	// Create branch
	branchName := args[1]
	if strings.HasPrefix(branchName, "-") {
		return "", fmt.Errorf("unknown switch `c' configuration: %s", branchName)
	}

	headRef, err := s.Repo.Head()
	if err != nil {
		return "", fmt.Errorf("cannot create branch: %v (maybe no commits yet?)", err)
	}

	// Create new reference
	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	newRef := plumbing.NewHashReference(refName, headRef.Hash())

	if err := s.Repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	return "Created branch " + branchName, nil
}

func (c *BranchCommand) Help() string {
	return "usage: git branch [-d] [<branchname>]\n\nList, create, or delete branches."
}
