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

	// Usage: git cherry-pick <commit>...
	// Usage: git cherry-pick <start>..<end>

	cmdArgs := args[1:]
	if len(cmdArgs) == 0 {
		return "", fmt.Errorf("usage: git cherry-pick <commit>...")
	}

	var commitsToPick []*object.Commit

	// Parse arguments (Support mixed ranges and single commits)
	for _, arg := range cmdArgs {
		if strings.Contains(arg, "..") {
			// Range detected: A..B
			parts := strings.SplitN(arg, "..", 2)
			startRev, endRev := parts[0], parts[1]

			if startRev == "" || endRev == "" {
				return "", fmt.Errorf("malformed range: %s", arg)
			}

			startHash, err := c.resolveRevision(repo, startRev)
			if err != nil {
				return "", fmt.Errorf("invalid revision '%s': %v", startRev, err)
			}
			endHash, err := c.resolveRevision(repo, endRev)
			if err != nil {
				return "", fmt.Errorf("invalid revision '%s': %v", endRev, err)
			}

			endCommit, err := repo.CommitObject(*endHash)
			if err != nil {
				return "", err
			}

			iter := endCommit
			var rangeCommits []*object.Commit
			foundStart := false

			// Walk back from End until Start
			for {
				if iter.Hash == *startHash {
					foundStart = true
					break
				}
				rangeCommits = append(rangeCommits, iter)

				if iter.NumParents() == 0 {
					break
				}
				p, err := iter.Parent(0)
				if err != nil {
					return "", fmt.Errorf("failed to traverse history: %v", err)
				}
				iter = p
			}

			if !foundStart {
				return "", fmt.Errorf("fatal: start revision '%s' is not an ancestor of '%s'", startRev, endRev)
			}

			// Add in correct order (Oldest -> Newest)
			for i := len(rangeCommits) - 1; i >= 0; i-- {
				commitsToPick = append(commitsToPick, rangeCommits[i])
			}

		} else {
			// Single commit
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
	}

	w, _ := repo.Worktree()
	headRef, _ := repo.Head()

	pickedCount := 0
	for _, commitToPick := range commitsToPick {
		// Prepare for 3-way merge
		// Base: Parent of the commit we are picking
		// Ours: Current HEAD
		// Theirs: The commit we are picking

		if commitToPick.NumParents() == 0 {
			// Picking a root commit? technically possible, treat base as empty.
			// For simplicity in this simulation, maybe just use ApplyCommit logic or error?
			// Let's defer to ApplyCommitChanges for root commits or handle empty base.
			// Current Merge3Way handles nil base.
		}

		// Get current HEAD (Ours)
		headRef, err := repo.Head()
		if err != nil {
			return "", err
		}
		oursCommit, err := repo.CommitObject(headRef.Hash())
		if err != nil {
			return "", err
		}

		var baseCommit *object.Commit
		if commitToPick.NumParents() > 0 {
			baseCommit, _ = commitToPick.Parent(0)
		}

		// Execute Merge
		err = git.Merge3Way(w, baseCommit, oursCommit, commitToPick)
		if err != nil {
			if err == git.ErrConflict {
				return "", fmt.Errorf("error: could not apply %s... %s\nhint: after resolving the conflicts, mark the corrected paths\nhint: with 'git add <paths>' or 'git rm <paths>'\nhint: and commit the result with 'git commit'", commitToPick.Hash.String()[:7], commitToPick.Message)
			}
			return "", fmt.Errorf("failed to cherry-pick %s: %v", commitToPick.Hash.String()[:7], err)
		}

		time.Sleep(10 * time.Millisecond)

		// Commit
		_, err = w.Commit(commitToPick.Message, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  commitToPick.Author.Name,
				Email: commitToPick.Author.Email,
				When:  time.Now(),
			},
			AllowEmptyCommits: true, // cherry-pick allows empty if content matches (usually requires --allow-empty but for simulation we are permissive)
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
