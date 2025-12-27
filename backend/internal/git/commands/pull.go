package commands

// pull.go - Simulated Git Pull Command
//
// Fetches from and integrates with another repository or a local branch.
// This is equivalent to git fetch + git merge in simulation.
// IMPORTANT: No actual network operations are performed.

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("pull", func() git.Command { return &PullCommand{} })
}

type PullCommand struct{}

func (c *PullCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// git pull = git fetch + git merge

	// git pull = git fetch + git merge

	isDryRun := false
	var cleanArgs []string

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-n", "--dry-run":
			isDryRun = true
		case "-h", "--help":
			return c.Help(), nil
		default:
			cleanArgs = append(cleanArgs, arg)
		}
	}

	// 1. Fetch
	fetchArgs := []string{"fetch"}
	if isDryRun {
		fetchArgs = append(fetchArgs, "--dry-run")
	}
	remoteName := "origin"
	if len(cleanArgs) > 0 {
		remoteName = cleanArgs[0]
		fetchArgs = append(fetchArgs, remoteName)
	}

	fetchCmd := &FetchCommand{}
	fetchOutput, err := fetchCmd.Execute(ctx, s, fetchArgs)
	if err != nil {
		return "", fmt.Errorf("pull (fetch failed): %w", err)
	}

	if isDryRun {
		return fmt.Sprintf("%s\n[dry-run] Pull would continue with merge/rebase.", fetchOutput), nil
	}

	// 2. Determine upstream branch to merge
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

	var mergeRefName string
	if len(cleanArgs) > 1 {
		// git pull origin main
		branchName := cleanArgs[1]
		mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName)
	} else {
		if headRef.Name().IsBranch() {
			currentBranch := headRef.Name().Short()
			mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", remoteName, currentBranch)
		} else {
			return "", fmt.Errorf("HEAD is detached, please specify remote ref to merge")
		}
	}

	// 3. Merge (Fast-Forward only for now)
	mergeRef, err := repo.Reference(plumbing.ReferenceName(mergeRefName), true)
	if err != nil {
		return fmt.Sprintf("%s\n(merge skipped: ref %s not found)", fetchOutput, mergeRefName), nil
	}

	headHash := headRef.Hash()
	targetHash := mergeRef.Hash()

	// 3. Merge Flow
	// Check for Fast-Forward first (optimization)
	headCommit, err := repo.CommitObject(headHash)
	if err != nil {
		return "", err
	}
	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return "", err
	}

	isFF, err := git.IsFastForward(repo, headHash, targetHash)
	if err != nil {
		return "", err
	}

	if isFF {
		// Perform FF Merge (Update HEAD ref)
		newRef := plumbing.NewHashReference(headRef.Name(), targetHash)
		err = repo.Storer.SetReference(newRef)
		if err != nil {
			return "", err
		}

		// Update Working Tree
		w, wErr := repo.Worktree()
		if wErr != nil {
			return "", wErr
		}
		err = w.Reset(&gogit.ResetOptions{
			Commit: targetHash,
			Mode:   gogit.HardReset,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update worktree: %w", err)
		}

		return fmt.Sprintf("%s\nUpdating %s..%s\nFast-forward", fetchOutput, headHash.String()[:7], targetHash.String()[:7]), nil
	}

	// 4. True Merge (3-Way)
	// Find Merge Base
	mergeBases, err := headCommit.MergeBase(targetCommit)
	if err != nil {
		return "", fmt.Errorf("failed to calculate merge base: %w", err)
	}
	if len(mergeBases) == 0 {
		return "", fmt.Errorf("refusing to merge unrelated histories")
	}
	baseCommit := mergeBases[0]

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// Perform 3-Way Merge
	err = git.Merge3Way(w, baseCommit, headCommit, targetCommit)
	if err != nil {
		if err == git.ErrConflict {
			return fmt.Sprintf("%s\nCONFLICT (content): Merge conflict detected.\nAutomatic merge failed; fix conflicts and then commit the result.", fetchOutput), nil
		}
		return "", fmt.Errorf("merge failed: %w", err)
	}

	// Success: Stage and Commit
	// We assume Merge3Way updated the worktree files. Now we stage them.
	// In a real git, we'd only stage changed files, but Add(".") is acceptable for simulation.
	_, err = w.Add(".")
	if err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	message := fmt.Sprintf("Merge branch '%s' into %s", mergeRefName, headRef.Name().Short())
	// Clean up ref name for message
	if len(cleanArgs) > 1 {
		message = fmt.Sprintf("Merge branch '%s' of %s into %s", cleanArgs[1], remoteName, headRef.Name().Short())
	}

	mergeCommit, err := w.Commit(message, &gogit.CommitOptions{
		Parents:   []plumbing.Hash{headHash, targetHash},
		Author:    git.GetDefaultSignature(),
		Committer: git.GetDefaultSignature(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create merge commit: %w", err)
	}

	return fmt.Sprintf("%s\nMerge made by the 'ort' strategy.\n%s", fetchOutput, mergeCommit.String()[:7]), nil
}

func (c *PullCommand) Help() string {
	return `📘 GIT-PULL (1)                                         Git Manual

 🚀 NAME
    git-pull - リモートから取得し、統合する

 📋 SYNOPSIS
    git pull [<remote>] [<branch>]

 💡 DESCRIPTION
    ` + "`" + `git fetch` + "`" + ` と ` + "`" + `git merge` + "`" + ` を一度に行うコマンドです。
    リモートの変更を取得し、現在のブランチにマージします。
`
}

// isFastForward moved to utils.go
