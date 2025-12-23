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

	isDryRun := false
	isHelp := false
	var cleanArgs []string
	for i, arg := range args {
		if i == 0 {
			continue
		}
		switch arg {
		case "-n", "--dry-run":
			isDryRun = true
		case "-h", "--help":
			isHelp = true
		default:
			cleanArgs = append(cleanArgs, arg)
		}
	}

	if isHelp {
		return c.Help(), nil
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

	if headHash == targetHash {
		return fmt.Sprintf("%s\nAlready up to date.", fetchOutput), nil
	}

	isFF, err := isFastForward(repo, headHash, targetHash)
	if err != nil {
		return "", err
	}

	if isFF {
		// Perform FF Merge
		name := headRef.Name()
		newRef := plumbing.NewHashReference(name, targetHash)
		err = repo.Storer.SetReference(newRef)
		if err != nil {
			return "", err
		}

		w, err := repo.Worktree()
		if err != nil {
			return "", err
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
	return `usage: git pull [options] [<remote>] [<branch>]

Options:
    -n, --dry-run     dry run
    --help            display this help message

Fetch from and integrate with another repository or a local branch.
Note: This is a simulated pull from virtual remotes.
`
}

// isFastForward moved to utils.go
