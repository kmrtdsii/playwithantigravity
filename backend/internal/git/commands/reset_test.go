package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestResetCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-reset")

	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"})

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Create commits
	createCommit := func(msg string) {
		f, _ := w.Filesystem.Create("file.txt")
		f.Write([]byte(msg))
		f.Close()
		w.Add("file.txt")
		w.Commit(msg, &gogit.CommitOptions{Author: &object.Signature{Name: "Me", When: time.Now()}})
	}

	createCommit("first")
	createCommit("second")
	createCommit("third")

	cmd := &ResetCommand{}

	t.Run("Soft Reset", func(t *testing.T) {
		// Reset to HEAD~1
		res, err := cmd.Execute(context.Background(), s, []string{"reset", "--soft", "HEAD~1"})
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		if !strings.Contains(res, "HEAD is now at") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		c, _ := repo.CommitObject(head.Hash())
		if c.Message != "second" {
			t.Errorf("Expected HEAD at 'second', got '%s'", c.Message)
		}

		// Index should still have content of 'third' (staged)
		// We can check status or just that worktree file has content
	})

	t.Run("Hard Reset", func(t *testing.T) {
		_, err := cmd.Execute(context.Background(), s, []string{"reset", "--hard", "HEAD~1"})
		if err != nil {
			t.Fatalf("Reset hard failed: %v", err)
		}

		head, _ := repo.Head()
		c, _ := repo.CommitObject(head.Hash())
		if c.Message != "first" {
			t.Errorf("Expected HEAD at 'first', got '%s'", c.Message)
		}
	})
}
