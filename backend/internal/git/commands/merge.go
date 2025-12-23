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

func (c *MergeCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	w, _ := repo.Worktree()
	// Parse Flags
	targetName := ""
	squash := false
	isDryRun := false

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--squash":
			squash = true
		case "--dry-run", "-n":
			isDryRun = true
		case "--help", "-h":
			return c.Help(), nil
		default:
			if targetName == "" {
				targetName = arg
			}
		}
	}

	if targetName == "" {
		return "", fmt.Errorf("usage: git merge [--squash] [--dry-run] <branch>")
	}

	// 1. Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// 2. Resolve Target
	// Try resolving as branch first
	targetRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+targetName), true)
	var targetHash plumbing.Hash
	if err == nil {
		targetHash = targetRef.Hash()
	} else {
		// Try as hash
		targetHash = plumbing.NewHash(targetName)
	}

	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return "", fmt.Errorf("merge: %s - not something we can merge", targetName)
	}

	// Update ORIG_HEAD before any merge operation
	s.UpdateOrigHead()

	// --- SQUASH HANDLING ---
	if squash {
		if isDryRun {
			return fmt.Sprintf("[dry-run] Would squash-merge %s into current branch (worktree would be updated but no commit created)", targetName), nil
		}
		// 1. Apply changes from target to worktree (Simplified: Overwrite/Add from Target)
		tree, err := targetCommit.Tree()
		if err != nil {
			return "", err
		}

		err = tree.Files().ForEach(func(f *object.File) error {
			// Write content
			content, err := f.Contents()
			if err != nil {
				return err
			}

			// Identify path
			path := f.Name

			// Write to FS
			fsFile, err := w.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			defer fsFile.Close()
			fsFile.Write([]byte(content))

			// Stage
			_, err = w.Add(path)
			return err
		})
		if err != nil {
			return "", err
		}

		// 2. Do NOT commit.
		return "Squash merge -- not committed", nil
	}

	// 3. Analyze Ancestry
	base, err := targetCommit.MergeBase(headCommit)
	if err == nil && len(base) > 0 {
		// Check for "Already up to date"
		if base[0].Hash == targetCommit.Hash {
			return "Already up to date.", nil
		}

		// Check for Fast-Forward
		if base[0].Hash == headCommit.Hash {
			if isDryRun {
				return fmt.Sprintf("[dry-run] Would perform fast-forward merge of %s", targetName), nil
			}
			if headRef.Name().IsBranch() {
				// We are on a branch. Use Reset --hard
				s.UpdateOrigHead()

				err = w.Reset(&gogit.ResetOptions{
					Commit: targetCommit.Hash,
					Mode:   gogit.HardReset,
				})
				if err != nil {
					return "", err
				}

				return fmt.Sprintf("Updating %s..%s\nFast-forward", headCommit.Hash.String()[:7], targetCommit.Hash.String()[:7]), nil
			} else {
				// Detached HEAD
				s.UpdateOrigHead()

				err = w.Checkout(&gogit.CheckoutOptions{
					Hash: targetCommit.Hash,
				})
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("Fast-forward to %s", targetName), nil
			}
		}
	}

	if isDryRun {
		s.PotentialCommits = []git.Commit{
			{
				ID:             "sim-merge",
				Message:        fmt.Sprintf("Merge branch '%s' (simulation)", targetName),
				ParentID:       headCommit.Hash.String(),
				SecondParentID: targetCommit.Hash.String(),
				Timestamp:      time.Now().Format(time.RFC3339),
			},
		}
		return fmt.Sprintf("[dry-run] Would create merge commit for %s (strategy 'ort')", targetName), nil
	}

	// 4. Merge Commit
	msg := fmt.Sprintf("Merge branch '%s'", targetName)
	parents := []plumbing.Hash{headCommit.Hash, targetCommit.Hash}

	s.UpdateOrigHead()

	newCommitHash, err := w.Commit(msg, &gogit.CommitOptions{
		Parents: parents,
		Author: &object.Signature{
			Name:  "User",
			Email: "user@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Merge made by the 'ort' strategy.\n %s", newCommitHash.String()), nil
}

func (c *MergeCommand) Help() string {
	return `usage: git merge [options] <branch>

Options:
    --squash          squash merge (apply changes but do not commit)
    -n, --dry-run     dry run
    --help            display this help message

Join two or more development histories together.
`
}
