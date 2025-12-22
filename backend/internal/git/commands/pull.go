package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("pull", func() git.Command { return &PullCommand{} })
}

type PullCommand struct{}

func (c *PullCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// git pull = git fetch + git merge

	// 1. Fetch
	// Construct args for FetchCommand
	// git pull [remote] [branch] -> fetch [remote]

	fetchArgs := []string{"fetch"}
	remoteName := "origin"
	if len(args) > 1 {
		remoteName = args[1]
		fetchArgs = append(fetchArgs, remoteName)
	}

	fetchCmd := &FetchCommand{}
	fetchOutput, err := fetchCmd.Execute(ctx, s, fetchArgs)
	if err != nil {
		return "", fmt.Errorf("pull (fetch failed): %w", err)
	}

	// 2. Determine upstream branch to merge
	// Default: matches current branch name?
	// or from args? git pull origin main -> merge origin/main

	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	// Logic to finding what to merge
	var mergeRefName string

	if len(args) > 2 {
		// git pull origin main
		branchName := args[2]
		mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName)
	} else {
		// git pull (implicit) or git pull origin
		// derive from current branch
		if headRef.Name().IsBranch() {
			currentBranch := headRef.Name().Short()
			mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", remoteName, currentBranch) // Assuming 1:1 mapping
		} else {
			return "", fmt.Errorf("HEAD is detached, please specify remote ref to merge")
		}
	}

	// 3. Merge (Fast-Forward only for now)
	// We check if we can FF.

	// Resolve mergeRef
	mergeRef, err := repo.Reference(plumbing.ReferenceName(mergeRefName), true)
	if err != nil {
		// If ref doesn't exist, maybe it wasn't fetched or typo
		return fmt.Sprintf("%s\n(merge skipped: ref %s not found)", fetchOutput, mergeRefName), nil
	}

	// Check Ancestry
	// If HEAD is ancestor of MergeRef -> Fast Forward
	// If MergeRef is ancestor of HEAD -> Already up to date
	// If Diverged -> Conflict (not implemented)

	headHash := headRef.Hash()
	targetHash := mergeRef.Hash()

	if headHash == targetHash {
		return fmt.Sprintf("%s\nAlready up to date.", fetchOutput), nil
	}

	isFF, err := isFastForward(repo, headHash, targetHash)
	if err != nil {
		return "", err
	}

	if isFF {
		// Perform FF Merge
		// Update HEAD to targetHash
		// And update Worktree?

		// Update HEAD Ref
		name := headRef.Name()
		newRef := plumbing.NewHashReference(name, targetHash)
		err = repo.Storer.SetReference(newRef)
		if err != nil {
			return "", err
		}

		// Update Worktree to match new HEAD
		// Naive: Checkout?
		w, err := repo.Worktree()
		if err != nil {
			return "", err
		}

		err = w.Reset(&gogit.ResetOptions{
			Commit: targetHash,
			Mode:   gogit.HardReset, // For Simulation, Hard Reset is easiest Update
		})
		if err != nil {
			return "", fmt.Errorf("failed to update worktree: %w", err)
		}

		return fmt.Sprintf("%s\nUpdating %s..%s\nFast-forward", fetchOutput, headHash.String()[:7], targetHash.String()[:7]), nil
	}

	// Check reverse (Already up to date check more formally)
	isUpToDate, err := isFastForward(repo, targetHash, headHash)
	if err != nil {
		return "", err
	}
	if isUpToDate {
		return fmt.Sprintf("%s\nAlready up to date.", fetchOutput), nil
	}

	return fmt.Sprintf("%s\nfatal: Not possible to fast-forward, aborting. (Merge strategy not implemented in simulation yet)", fetchOutput), nil
}

func (c *PullCommand) Help() string {
	return "usage: git pull [remote] [branch]"
}

// isFastForward moved to utils.go
