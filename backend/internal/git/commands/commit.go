package commands

// commit.go - Simulated Git Commit Command
//
// Records changes to the repository by creating a new commit object.
// Supports -m (message), --amend, and --allow-empty flags.

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("commit", func() git.Command { return &CommitCommand{} })
}

type CommitCommand struct{}

func (c *CommitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := repo.Worktree()

	// Parse options
	msg := "Default commit message"
	amend := false
	allowEmpty := false

	// Naive arg parsing
	for i := 1; i < len(args); i++ {
		if args[i] == "-h" || args[i] == "--help" {
			return c.Help(), nil
		} else if args[i] == "-m" && i+1 < len(args) {
			msg = args[i+1]
			i++
		} else if args[i] == "--amend" {
			amend = true
		} else if args[i] == "--allow-empty" {
			allowEmpty = true
		}
	}

	if amend {
		// Amend logic
		headRef, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("cannot amend without HEAD: %v", err)
		}
		headCommit, err := repo.CommitObject(headRef.Hash())
		if err != nil {
			return "", err
		}

		parents := headCommit.ParentHashes

		// Reuse message if not provided explicitly in args
		// Simple check if -m was present
		isMsgProvided := false
		for i := 1; i < len(args); i++ {
			if args[i] == "-m" {
				isMsgProvided = true
				break
			}
		}
		if !isMsgProvided {
			msg = headCommit.Message
		}

		s.UpdateOrigHead()

		newCommitHash, err := w.Commit(msg, &gogit.CommitOptions{
			Parents: parents,
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
			AllowEmptyCommits: true, // Amending should always be allowed even if no changes
		})
		if err != nil {
			return "", err
		}
		s.RecordReflog("commit (amend): " + strings.Split(msg, "\n")[0])

		return fmt.Sprintf("Commit amended: %s", newCommitHash.String()), nil
	}

	// Normal commit
	commit, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "User",
			Email: "user@example.com",
			When:  time.Now(),
		},
		AllowEmptyCommits: allowEmpty,
	})
	if err != nil {
		// Add helpful hint for empty commit error
		if strings.Contains(err.Error(), "clean") || strings.Contains(err.Error(), "nothing to commit") {
			return "", fmt.Errorf("%v\nhint: Use 'git commit --allow-empty -m <message>' to create an empty commit", err)
		}
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("commit: %s", strings.Split(msg, "\n")[0]))
	return fmt.Sprintf("Commit created: %s", commit.String()), nil
}

func (c *CommitCommand) Help() string {
	return `usage: git commit [options]

Options:
    -m <msg>          use the given <msg> as the commit message
    --amend           amend the previous commit
    --allow-empty     allow creating a commit with no changes
    --help            display this help message

Record changes to the repository.
`
}
