package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("cherry-pick", func() git.Command { return &CherryPickCommand{} })
}

type CherryPickCommand struct{}

func (c *CherryPickCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Simple parser for cherry-pick
	// Usage: git cherry-pick <commit>...
	// Usage: git cherry-pick <start>..<end>

	cmdArgs := args[1:]
	if len(cmdArgs) == 0 {
		return "", fmt.Errorf("usage: git cherry-pick <commit>...")
	}

	// We assume one argument for range or multiple for single commits?
	// The user specifically asked for range A..B.
	// Standard git allows mixing. For now let's handle the first arg as potentially a range.

	arg := cmdArgs[0]
	var commitsToPick []*object.Commit

	if strings.Contains(arg, "..") {
		// Range detected
		parts := strings.SplitN(arg, "..", 2)
		startRev := parts[0]
		endRev := parts[1]

		if startRev == "" || endRev == "" {
			return "", fmt.Errorf("malformed range: %s", arg)
		}

		// Resolve revisions
		startHash, err := c.resolveRevision(repo, startRev)
		if err != nil {
			return "", fmt.Errorf("invalid revision '%s': %v", startRev, err)
		}
		endHash, err := c.resolveRevision(repo, endRev)
		if err != nil {
			return "", fmt.Errorf("invalid revision '%s': %v", endRev, err)
		}

		// Traverse from End to Start (exclusive)
		// A..B means (A, B]

		endCommit, err := repo.CommitObject(*endHash)
		if err != nil {
			return "", err
		}

		iter := endCommit
		var rangeCommits []*object.Commit
		foundStart := false

		for {
			if iter.Hash == *startHash {
				foundStart = true
				break
			}
			rangeCommits = append(rangeCommits, iter)

			if iter.NumParents() == 0 {
				break
			}
			// Linear assumption fallback
			p, err := iter.Parent(0)
			if err != nil {
				return "", fmt.Errorf("failed to traverse history: %v", err)
			}
			iter = p
		}

		if !foundStart {
			// If we didn't find start, maybe they are divergent?
			// git cherry-pick A..B usually requires ancestry or it behaves like log A..B (reachable from B not A)
			// Efficiently implementing A..B for complex graphs is hard.
			// Assuming linear or simple error.
			return "", fmt.Errorf("fatal: start revision '%s' is not an ancestor of '%s'", startRev, endRev)
		}

		// rangeCommits are in reverse order (End -> Start child)
		// Reverse them to Apply in order
		for i := len(rangeCommits) - 1; i >= 0; i-- {
			commitsToPick = append(commitsToPick, rangeCommits[i])
		}

	} else {
		// Single commit or list
		h, err := c.resolveRevision(repo, arg)
		if err != nil {
			return "", fmt.Errorf("bad revision '%s'", arg)
		}
		commit, err := repo.CommitObject(*h)
		if err != nil {
			return "", err
		}
		commitsToPick = append(commitsToPick, commit)
	}

	w, _ := repo.Worktree()
	headRef, _ := repo.Head() // To record in result message

	pickedCount := 0
	for _, c := range commitsToPick {
		// Apply changes using shared helper
		if err := git.ApplyCommitChanges(w, c); err != nil {
			return "", fmt.Errorf("failed to apply commit %s: %v", c.Hash.String()[:7], err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure unique timestamp

		// Commit with same message and author
		_, err := w.Commit(c.Message, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  c.Author.Name,
				Email: c.Author.Email,
				When:  time.Now(), // Committer time is now
			},
			AllowEmptyCommits: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to commit: %v", err)
		}
		pickedCount++
	}

	return fmt.Sprintf("Cherry-pick successful. Picked %d commits to %s.", pickedCount, headRef.Name().Short()), nil
}

// resolveRevision delegates to the shared git.ResolveRevision helper
func (c *CherryPickCommand) resolveRevision(repo *gogit.Repository, rev string) (*plumbing.Hash, error) {
	return git.ResolveRevision(repo, rev)
}

func (c *CherryPickCommand) Help() string {
	return `usage: git cherry-pick <commit>...
       git cherry-pick <start>..<end>`
}
