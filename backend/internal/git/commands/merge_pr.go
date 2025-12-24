package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("merge-pr", func() git.Command { return &MergePRCommand{} })
}

type MergePRCommand struct{}

func (c *MergePRCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// Usage: merge-pr <pr-id> <remote-name>
	// args[0] is "merge-pr"

	if len(args) < 3 {
		return "", fmt.Errorf("usage: merge-pr <pr-id> <remote-name>")
	}

	prIDStr := args[1]
	remoteName := args[2]

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid pr id: %v", err)
	}

	// Access SessionManager via Session
	sm := s.Manager

	// Logic moved from actions.go
	sm.Lock() // safe to lock whole manager for this operation
	defer sm.Unlock()

	var pr *git.PullRequest
	for _, p := range sm.PullRequests {
		if p.ID == prID {
			pr = p
			break
		}
	}
	if pr == nil {
		return "", fmt.Errorf("pull request %d not found", prID)
	}

	if pr.State != "OPEN" {
		return "", fmt.Errorf("pull request is not open")
	}

	repo, ok := sm.SharedRemotes[remoteName]
	if !ok {
		return "", fmt.Errorf("remote %s not found", remoteName)
	}

	// Resolve references
	baseRefName := plumbing.ReferenceName("refs/heads/" + pr.BaseRef)
	headRefName := plumbing.ReferenceName("refs/heads/" + pr.HeadRef)

	baseRef, err := repo.Reference(baseRefName, true)
	if err != nil {
		return "", fmt.Errorf("base branch %s not found: %w", pr.BaseRef, err)
	}
	headRef, err := repo.Reference(headRefName, true)
	if err != nil {
		return "", fmt.Errorf("head branch %s not found: %w", pr.HeadRef, err)
	}

	// 1. Get HEAD Commit (from Base)
	baseCommit, err := repo.CommitObject(baseRef.Hash())
	if err != nil {
		return "", err
	}
	// 2. Get Merge Commit (from Head)
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// 3. Create Merge Commit using "Theirs" tree (Head's tree)
	mergeCommit := &object.Commit{
		Author: object.Signature{
			Name:  "Merge Bot",
			Email: "bot@gitgym.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "Merge Bot",
			Email: "bot@gitgym.com",
			When:  time.Now(),
		},
		Message:  fmt.Sprintf("Merge pull request #%d from %s\n\n%s", prID, pr.HeadRef, pr.Title),
		TreeHash: headCommit.TreeHash, // Taking the tree from the feature branch
		ParentHashes: []plumbing.Hash{
			baseCommit.Hash,
			headCommit.Hash,
		},
	}

	obj := repo.Storer.NewEncodedObject()
	if encodeErr := mergeCommit.Encode(obj); encodeErr != nil {
		return "", encodeErr
	}
	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	// Update Base Ref to point to new commit
	newRef := plumbing.NewHashReference(baseRefName, newHash)
	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	pr.State = "MERGED"
	return fmt.Sprintf("Merged PR #%d into %s", prID, pr.BaseRef), nil
}

func (c *MergePRCommand) Help() string {
	return "usage: merge-pr <pr-id> <remote-name>"
}
