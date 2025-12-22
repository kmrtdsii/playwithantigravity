package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("rebase", func() git.Command { return &RebaseCommand{} })
}

type RebaseCommand struct{}

func (c *RebaseCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git rebase <upstream>")
	}

	// Update ORIG_HEAD before rebase starts
	s.UpdateOrigHead()

	upstreamName := args[1]

	// 1. Resolve Upstream
	upstreamHash, err := repo.ResolveRevision(plumbing.Revision(upstreamName))
	if err != nil {
		return "", fmt.Errorf("invalid upstream '%s': %v", upstreamName, err)
	}
	upstreamCommit, err := repo.CommitObject(*upstreamHash)
	if err != nil {
		return "", err
	}

	// 2. Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// 3. Find Merge Base
	mergeBases, err := upstreamCommit.MergeBase(headCommit)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base: %v", err)
	}
	if len(mergeBases) == 0 {
		return "", fmt.Errorf("no common ancestor found")
	}
	base := mergeBases[0]

	if base.Hash == headCommit.Hash {
		return "Current branch is up to date.", nil
	}
	if base.Hash == upstreamCommit.Hash {
		return "Current branch is up to date (or ahead of upstream).", nil
	}

	// 4. Collect commits to replay (base..HEAD]
	var commitsToReplay []*object.Commit
	iter := headCommit
	for iter.Hash != base.Hash {
		commitsToReplay = append(commitsToReplay, iter)
		if iter.NumParents() == 0 {
			break
		}
		p, err := iter.Parent(0)
		if err != nil {
			return "", fmt.Errorf("failed to traverse parents: %v", err)
		}
		iter = p
	}
	// Reverse order
	for i, j := 0, len(commitsToReplay)-1; i < j; i, j = i+1, j-1 {
		commitsToReplay[i], commitsToReplay[j] = commitsToReplay[j], commitsToReplay[i]
	}

	// 5. Hard Reset to Upstream
	w, _ := repo.Worktree()
	if err := w.Reset(&gogit.ResetOptions{Commit: *upstreamHash, Mode: gogit.HardReset}); err != nil {
		return "", fmt.Errorf("failed to reset to upstream: %v", err)
	}

	// 6. Replay Commits (Cherry-pick)
	replayedCount := 0
	for _, c := range commitsToReplay {
		parent, _ := c.Parent(0)
		pTree, _ := parent.Tree()
		cTree, _ := c.Tree()
		patch, err := pTree.Patch(cTree)
		if err != nil {
			return "", fmt.Errorf("failed to compute patch: %v", err)
		}

		for _, fp := range patch.FilePatches() {
			from, to := fp.Files()
			if to == nil {
				if from != nil {
					w.Filesystem.Remove(from.Path())
				}
				continue
			}
			path := to.Path()
			file, err := c.File(path)
			if err != nil {
				continue
			}
			content, err := file.Contents()
			if err != nil {
				continue
			}

			f, _ := w.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			f.Write([]byte(content))
			f.Close()
			w.Add(path)
		}

		// Ensure timestamp distinctness
		time.Sleep(10 * time.Millisecond)

		_, err = w.Commit(c.Message, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
			AllowEmptyCommits: true, // Replaying commits should allow empty ones if they existed
		})
		if err != nil {
			return "", fmt.Errorf("failed to commit replayed change: %v", err)
		}
		replayedCount++
	}

	s.RecordReflog(fmt.Sprintf("rebase: finished rebase onto %s", upstreamName))
	return fmt.Sprintf("Successfully rebased and updated %s.\nReplayed %d commits.", headRef.Name().Short(), replayedCount), nil
}

func (c *RebaseCommand) Help() string {
	return "usage: git rebase <upstream>"
}
