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

// Ensure CherryPickCommand implements git.Command
var _ git.Command = (*CherryPickCommand)(nil)

type CherryPickOptions struct {
	Args []string
}

func (c *CherryPickCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	commits, err := c.resolveCommits(repo, opts.Args)
	if err != nil {
		return "", err
	}

	return c.executeCherryPick(s, repo, commits)
}

func (c *CherryPickCommand) parseArgs(args []string) (*CherryPickOptions, error) {
	cmdArgs := args[1:]
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("usage: git cherry-pick <commit>")
	}
	// For now, no flags supported, just assume all are commit args
	return &CherryPickOptions{Args: cmdArgs}, nil
}

func (c *CherryPickCommand) resolveCommits(repo *gogit.Repository, args []string) ([]*object.Commit, error) {
	var commitsToPick []*object.Commit

	for _, arg := range args {
		if strings.Contains(arg, "..") {
			// Range detected: A..B
			parts := strings.SplitN(arg, "..", 2)
			startRev, endRev := parts[0], parts[1]

			if startRev == "" || endRev == "" {
				return nil, fmt.Errorf("malformed range: %s", arg)
			}

			startHash, err := c.resolveRevision(repo, startRev)
			if err != nil {
				return nil, fmt.Errorf("invalid revision '%s': %v", startRev, err)
			}
			endHash, err := c.resolveRevision(repo, endRev)
			if err != nil {
				return nil, fmt.Errorf("invalid revision '%s': %v", endRev, err)
			}

			endCommit, err := repo.CommitObject(*endHash)
			if err != nil {
				return nil, err
			}

			iter := endCommit
			var rangeCommits []*object.Commit
			foundStart := false

			// Walk back from End until Start
			// Safety limit for simulation?
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
					return nil, fmt.Errorf("failed to traverse history: %v", err)
				}
				iter = p
			}

			if !foundStart {
				return nil, fmt.Errorf("fatal: start revision '%s' is not an ancestor of '%s'", startRev, endRev)
			}

			// Add in correct order (Oldest -> Newest)
			for i := len(rangeCommits) - 1; i >= 0; i-- {
				commitsToPick = append(commitsToPick, rangeCommits[i])
			}

		} else {
			// Single commit
			h, err := c.resolveRevision(repo, arg)
			if err != nil {
				return nil, fmt.Errorf("bad revision '%s'", arg)
			}
			commit, err := repo.CommitObject(*h)
			if err != nil {
				return nil, err
			}
			commitsToPick = append(commitsToPick, commit)
		}
	}
	return commitsToPick, nil
}

func (c *CherryPickCommand) executeCherryPick(_ *git.Session, repo *gogit.Repository, commitsToPick []*object.Commit) (string, error) {
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	pickedCount := 0
	for _, commitToPick := range commitsToPick {
		// Prepare for 3-way merge
		// Base: Parent of the commit we are picking
		// Ours: Current HEAD
		// Theirs: The commit we are picking

		// if commitToPick.NumParents() == 0 {
		// 	// Picking a root commit - not handled in basic flow yet
		// }

		// Get current HEAD (Ours)
		headRef, err = repo.Head() // Update HEAD ref in each iteration as it moves
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
	return `📘 GIT-CHERRY-PICK (1)                                  Git Manual

 💡 DESCRIPTION
    ・別のブランチにある「特定のコミット」だけをコピーして取り込む
    ・指定したコミットの変更を、現在のブランチに適用する

 📋 SYNOPSIS
    git cherry-pick <commit>...
    git cherry-pick <start>..<end>

 ⚙️  COMMON OPTIONS
    <commit>...
        適用したいコミットのハッシュ。複数指定可能。

    <start>..<end>
        コミットの範囲を指定します（startを含まず、endまで）。

 🛠  EXAMPLES
    1. 特定のコミットを適用
       $ git cherry-pick e5a3b21

    2. 範囲適用
       $ git cherry-pick A..B

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-cherry-pick
`
}
