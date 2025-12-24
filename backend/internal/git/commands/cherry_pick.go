package commands

import (
	"context"
	"fmt"
	"os"
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
		// Apply changes
		// Calculate patch from Parent -> C
		var pTree *object.Tree
		if c.NumParents() > 0 {
			parent, _ := c.Parent(0)
			pTree, _ = parent.Tree()
		}

		cTree, _ := c.Tree()

		var patch *object.Patch
		if pTree == nil {
			// Root commit... handle files
			files, _ := c.Files()
			files.ForEach(func(f *object.File) error {
				wFile, _ := w.Filesystem.OpenFile(f.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
				content, _ := f.Contents()
				wFile.Write([]byte(content))
				wFile.Close()
				w.Add(f.Name)
				return nil
			})
		} else {
			var err error
			patch, err = pTree.Patch(cTree)
			if err != nil {
				return "", fmt.Errorf("failed to patch: %v", err)
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
		}

		time.Sleep(10 * time.Millisecond) // Ensure unique timestamp

		// Commit with same message and author
		_, err := w.Commit(c.Message, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  c.Author.Name,
				Email: c.Author.Email,
				When:  time.Now(), // Committer time is now
			},
			AllowEmptyCommits: true, // cherry-pick usually allows empty if it becomes empty? No, usually stops.
			// But for simulation, allow.
		})
		if err != nil {
			return "", fmt.Errorf("failed to commit: %v", err)
		}
		pickedCount++
	}

	return fmt.Sprintf("Cherry-pick successful. Picked %d commits to %s.", pickedCount, headRef.Name().Short()), nil
}

// Duplicate of RebaseCommand.resolveRevision for now
func (c *CherryPickCommand) resolveRevision(repo *gogit.Repository, rev string) (*plumbing.Hash, error) {
	h, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err == nil {
		return h, nil
	}
	if len(rev) >= 4 && len(rev) < 40 {
		cIter, err := repo.CommitObjects()
		if err == nil {
			var match *plumbing.Hash
			found := false
			cIter.ForEach(func(c *object.Commit) error {
				if len(c.Hash.String()) >= len(rev) && c.Hash.String()[:len(rev)] == rev {
					if found {
						return fmt.Errorf("ambiguous short hash")
					}
					match = &c.Hash
					found = true
				}
				return nil
			})
			if found {
				return match, nil
			}
		}
	}
	return nil, err
}

func (c *CherryPickCommand) Help() string {
	return `usage: git cherry-pick <commit>...
       git cherry-pick <start>..<end>`
}
