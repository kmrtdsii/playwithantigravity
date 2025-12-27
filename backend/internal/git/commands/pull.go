package commands

// pull.go - Simulated Git Pull Command
//
// Fetches from and integrates with another repository or a local branch.
// This is equivalent to git fetch + git merge in simulation.
// IMPORTANT: No actual network operations are performed.

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("pull", func() git.Command { return &PullCommand{} })
}

type PullCommand struct{}

type PullOptions struct {
	DryRun bool
	Remote string
	Branch string // Optional
}

type pullContext struct {
	FetchOutput  string
	Repo         *gogit.Repository
	HeadRef      *plumbing.Reference
	MergeRef     *plumbing.Reference // The remote ref to merge
	MergeRefName string
}

func (c *PullCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// 1. Parse Args
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	// 2. Fetch (Delegate to FetchCommand)
	fetchOutput, err := c.executeFetch(ctx, s, opts)
	if err != nil {
		return "", fmt.Errorf("pull (fetch failed): %w", err)
	}

	if opts.DryRun {
		return fmt.Sprintf("%s\n[dry-run] Pull would continue with merge/rebase.", fetchOutput), nil
	}

	// 3. Resolve Context (Identify Merge Target)
	pCtx, err := c.resolveContext(s, opts, fetchOutput)
	if err != nil {
		return "", err
	}

	// 4. Perform Merge
	return c.performPullMerge(s, pCtx)
}

func (c *PullCommand) parseArgs(args []string) (*PullOptions, error) {
	opts := &PullOptions{
		Remote: "origin",
	}
	var cleanArgs []string
	cmdArgs := args[1:]

	for _, arg := range cmdArgs {
		switch arg {
		case "-n", "--dry-run":
			opts.DryRun = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if strings.HasPrefix(arg, "-") {
				// ignore
			} else {
				cleanArgs = append(cleanArgs, arg)
			}
		}
	}

	if len(cleanArgs) > 0 {
		opts.Remote = cleanArgs[0]
	}
	if len(cleanArgs) > 1 {
		opts.Branch = cleanArgs[1]
	}
	return opts, nil
}

func (c *PullCommand) executeFetch(ctx context.Context, s *git.Session, opts *PullOptions) (string, error) {
	fetchArgs := []string{"fetch"}
	if opts.DryRun {
		fetchArgs = append(fetchArgs, "--dry-run")
	}
	// Always fetch the specified remote (or default origin)
	fetchArgs = append(fetchArgs, opts.Remote)

	fetchCmd := &FetchCommand{}
	return fetchCmd.Execute(ctx, s, fetchArgs)
}

func (c *PullCommand) resolveContext(s *git.Session, opts *PullOptions, fetchOutput string) (*pullContext, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return nil, fmt.Errorf("fatal: not a git repository")
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, err
	}

	var mergeRefName string
	if opts.Branch != "" {
		// Explicit branch: git pull origin main
		mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", opts.Remote, opts.Branch)
	} else {
		// Implicit branch: derive from current branch
		if headRef.Name().IsBranch() {
			currentBranch := headRef.Name().Short()
			mergeRefName = fmt.Sprintf("refs/remotes/%s/%s", opts.Remote, currentBranch)
		} else {
			return nil, fmt.Errorf("HEAD is detached, please specify remote ref to merge")
		}
	}

	// Verify merge ref exists
	mergeRef, err := repo.Reference(plumbing.ReferenceName(mergeRefName), true)
	if err != nil {
		// If dry run, we might not care, but we are past dry run here.
		return nil, fmt.Errorf("ref %s not found (fetch might have failed to update it?)", mergeRefName)
	}

	return &pullContext{
		FetchOutput:  fetchOutput,
		Repo:         repo,
		HeadRef:      headRef,
		MergeRef:     mergeRef,
		MergeRefName: mergeRefName,
	}, nil
}

func (c *PullCommand) performPullMerge(_ *git.Session, pCtx *pullContext) (string, error) {
	// Need lock for repo operations?
	// s.GetRepo() returns pointer. Operations on repo are usually thread-safe or s is locked?
	// Legacy Execute locked s during resolve. Here we unlocked.
	// Should lock again or locking is fine?
	// Standard practice: if interacting with Session state, lock. Repo state might have its own locks.
	// But simple read/write to repo is fine.

	repo := pCtx.Repo
	headRef := pCtx.HeadRef
	mergeRef := pCtx.MergeRef

	headHash := headRef.Hash()
	targetHash := mergeRef.Hash()

	// Check Fast-Forward
	isFF, err := git.IsFastForward(repo, headHash, targetHash)
	if err != nil {
		return "", err
	}

	if isFF {
		// FF Update
		newRef := plumbing.NewHashReference(headRef.Name(), targetHash)
		err = repo.Storer.SetReference(newRef)
		if err != nil {
			return "", err
		}

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

		return fmt.Sprintf("%s\nUpdating %s..%s\nFast-forward", pCtx.FetchOutput, headHash.String()[:7], targetHash.String()[:7]), nil
	}

	// 3-Way Merge
	headCommit, err := repo.CommitObject(headHash)
	if err != nil {
		return "", err
	}
	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return "", err
	}

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

	err = git.Merge3Way(w, baseCommit, headCommit, targetCommit)
	if err != nil {
		if err == git.ErrConflict {
			return fmt.Sprintf("%s\nCONFLICT (content): Merge conflict detected.\nAutomatic merge failed; fix conflicts and then commit the result.", pCtx.FetchOutput), nil
		}
		return "", fmt.Errorf("merge failed: %w", err)
	}

	// Stage changes (simplified)
	_, err = w.Add(".")
	if err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	message := fmt.Sprintf("Merge branch '%s' into %s", pCtx.MergeRefName, headRef.Name().Short())

	mergeCommit, err := w.Commit(message, &gogit.CommitOptions{
		Parents:   []plumbing.Hash{headHash, targetHash},
		Author:    git.GetDefaultSignature(),
		Committer: git.GetDefaultSignature(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create merge commit: %w", err)
	}

	return fmt.Sprintf("%s\nMerge made by the 'ort' strategy.\n%s", pCtx.FetchOutput, mergeCommit.String()[:7]), nil
}

func (c *PullCommand) Help() string {
	return `📘 GIT-PULL (1)                                         Git Manual

 💡 DESCRIPTION
    ・リモートリポジトリから最新の変更をダウンロードする（fetch）
    ・ダウンロードした変更を現在のブランチに取り込む（merge）
    （fetch と merge を一度に行うコマンドです）

 📋 SYNOPSIS
    git pull [<remote>] [<branch>]
`
}

// isFastForward moved to utils.go
