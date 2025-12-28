package commands

// merge.go - Simulated Git Merge Command
//
// Joins two or more development histories together.
// Supports --squash and --dry-run flags.
// This is a simulation and creates merge commits in-memory.

import (
	"context"
	"fmt"
	"os"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("merge", func() git.Command { return &MergeCommand{} })
}

type MergeCommand struct{}

// Ensure MergeCommand implements git.Command
var _ git.Command = (*MergeCommand)(nil)

type MergeOptions struct {
	Target string
	Squash bool
	DryRun bool
	NoFF   bool
}

type mergeContext struct {
	TargetHash   plumbing.Hash
	TargetCommit *object.Commit
	HeadRef      *plumbing.Reference
	HeadCommit   *object.Commit
}

func (c *MergeCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	// 2. Resolve Context
	mCtx, err := c.resolveContext(repo, opts)
	if err != nil {
		return "", err
	}

	// Update ORIG_HEAD before any merge operation
	s.UpdateOrigHead()

	// 3. Execution
	return c.performMerge(s, repo, mCtx, opts)
}

func (c *MergeCommand) parseArgs(args []string) (*MergeOptions, error) {
	opts := &MergeOptions{}
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--squash":
			opts.Squash = true
		case "--no-ff":
			opts.NoFF = true
		case "--dry-run", "-n":
			opts.DryRun = true
		case "--help", "-h":
			return nil, fmt.Errorf("help requested")
		default:
			if opts.Target == "" {
				opts.Target = arg
			}
		}
	}

	if opts.Target == "" {
		return nil, fmt.Errorf("usage: git merge [--no-ff] [--squash] [--dry-run] <branch>")
	}
	return opts, nil
}

func (c *MergeCommand) resolveContext(repo *gogit.Repository, opts *MergeOptions) (*mergeContext, error) {
	// 1. Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return nil, err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, err
	}

	// 2. Resolve Target
	targetHashPtr, err := git.ResolveRevision(repo, opts.Target)
	if err != nil {
		return nil, fmt.Errorf("merge: %s - not something we can merge", opts.Target)
	}
	targetHash := *targetHashPtr

	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		// Try resolving as commit if not found? ResolveRevision usually returns hash.
		return nil, fmt.Errorf("merge: %s - not something we can merge (commit not found)", opts.Target)
	}

	return &mergeContext{
		TargetHash:   targetHash,
		TargetCommit: targetCommit,
		HeadRef:      headRef,
		HeadCommit:   headCommit,
	}, nil
}

func (c *MergeCommand) performMerge(s *git.Session, repo *gogit.Repository, mCtx *mergeContext, opts *MergeOptions) (string, error) {
	w, _ := repo.Worktree() // Error unlikely if repo exists

	// --- SQUASH HANDLING ---
	if opts.Squash {
		if opts.DryRun {
			return fmt.Sprintf("[dry-run] Would squash-merge %s into current branch (worktree would be updated but no commit created)", opts.Target), nil
		}
		// Apply changes from target (Simplified: Overwrite/Add)
		// Note: This logic overwrites files. Real squash merge handles deletions and conflicts.
		if err := c.applyTree(w, mCtx.TargetCommit); err != nil {
			return "", err
		}

		return "Squash merge -- not committed", nil
	}

	// 3. Analyze Ancestry
	base, err := mCtx.TargetCommit.MergeBase(mCtx.HeadCommit)
	if err == nil && len(base) > 0 {
		// Already up to date
		if base[0].Hash == mCtx.TargetCommit.Hash {
			return "Already up to date.", nil
		}

		// Fast-Forward Check
		// If base is head, then head is ancestor of target -> Fast Forward possible
		if base[0].Hash == mCtx.HeadCommit.Hash {
			// Fast-Forward allowed if NoFF is false
			if !opts.NoFF {
				if opts.DryRun {
					return fmt.Sprintf("[dry-run] Would perform fast-forward merge of %s", opts.Target), nil
				}
				s.UpdateOrigHead() // Ensure checked before mutation

				if mCtx.HeadRef.Name().IsBranch() {
					err = w.Reset(&gogit.ResetOptions{
						Commit: mCtx.TargetCommit.Hash,
						Mode:   gogit.HardReset,
					})
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("Updating %s..%s\nFast-forward", mCtx.HeadCommit.Hash.String()[:7], mCtx.TargetCommit.Hash.String()[:7]), nil
				} else {
					// Detached HEAD
					err = w.Checkout(&gogit.CheckoutOptions{
						Hash: mCtx.TargetCommit.Hash,
					})
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("Fast-forward to %s", opts.Target), nil
				}
			}
		}
	}

	if opts.DryRun {
		s.PotentialCommits = []git.Commit{
			{
				ID:             "sim-merge",
				Message:        fmt.Sprintf("Merge branch '%s' (simulation)", opts.Target),
				ParentID:       mCtx.HeadCommit.Hash.String(),
				SecondParentID: mCtx.TargetCommit.Hash.String(),
				Timestamp:      time.Now().Format(time.RFC3339),
			},
		}
		return fmt.Sprintf("[dry-run] Would create merge commit for %s (strategy 'ort')", opts.Target), nil
	}

	// 4. Merge Commit
	// Apply changes from target to worktree (Simulation: Overwrite/Add from Target "Theirs")
	if err := c.applyTree(w, mCtx.TargetCommit); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("Merge branch '%s'", opts.Target)
	parents := []plumbing.Hash{mCtx.HeadCommit.Hash, mCtx.TargetCommit.Hash}

	s.UpdateOrigHead()

	newCommitHash, err := w.Commit(msg, &gogit.CommitOptions{
		Parents:   parents,
		Author:    git.GetDefaultSignature(),
		Committer: git.GetDefaultSignature(),
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Merge made by the 'ort' strategy.\n %s", newCommitHash.String()), nil
}

func (c *MergeCommand) applyTree(w *gogit.Worktree, commit *object.Commit) error {
	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	return tree.Files().ForEach(func(f *object.File) error {
		content, contentErr := f.Contents()
		if contentErr != nil {
			return contentErr
		}
		path := f.Name
		fsFile, openErr := w.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if openErr != nil {
			return openErr
		}
		defer fsFile.Close()
		if _, writeErr := fsFile.Write([]byte(content)); writeErr != nil {
			return writeErr
		}
		_, err := w.Add(path)
		return err
	})
}

func (c *MergeCommand) Help() string {
	return `📘 GIT-MERGE (1)                                        Git Manual

 💡 DESCRIPTION
    ・別のブランチの変更を、現在のブランチに取り込む
    ・2つの異なる開発履歴を1つに統合する
    通常は「マージコミット」が自動的に作成されます。

 📋 SYNOPSIS
    git merge [--no-ff] [--squash] <branch>

 ⚙️  COMMON OPTIONS
    --no-ff
        Fast-forward 可能な場合でも、強制的にマージコミットを作成します。
        履歴上に「ここで統合した」という事実を明確に残したい場合に使います。

    --squash
        マージコミットを作成せず、変更内容のみをワーキングツリーに取り込みます。
        あとで自分でコミットする場合に使用します。

 🛠  PRACTICAL EXAMPLES
    1. 基本: featureブランチをマージ
       $ git merge feature/login

    2. 実践: マージコミットを必ず作る (Recommended)
       単なるポインタ移動(Fast-forward)ではなく、コミットを残します。
       $ git merge --no-ff feature/login
`
}
