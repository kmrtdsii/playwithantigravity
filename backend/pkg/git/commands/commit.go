package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("commit", func() git.Command { return &CommitCommand{} })
}

type CommitCommand struct{}

func (c *CommitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := s.Repo.Worktree()
	
	// Parse options
	msg := "Default commit message"
	amend := false

	// Naive arg parsing
	for i := 1; i < len(args); i++ {
		if args[i] == "-m" && i+1 < len(args) {
			msg = args[i+1]
			i++
		} else if args[i] == "--amend" {
			amend = true
		}
	}

	if amend {
		// Amend logic
		headRef, err := s.Repo.Head()
		if err != nil {
			return "", fmt.Errorf("cannot amend without HEAD: %v", err)
		}
		headCommit, err := s.Repo.CommitObject(headRef.Hash())
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
	})
	if err != nil {
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("commit: %s", strings.Split(msg, "\n")[0]))
	return fmt.Sprintf("Commit created: %s", commit.String()), nil
}

func (c *CommitCommand) Help() string {
	return "usage: git commit [-m <msg>] [--amend]\n\nRecord changes to the repository."
}
