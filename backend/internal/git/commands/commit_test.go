package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestCommitCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-commit")

	// Init
	// Init
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Helper to create and add file
	createAndAdd := func(name, content string) {
		f, _ := w.Filesystem.Create(name)
		f.Write([]byte(content))
		f.Close()
		w.Add(name)
	}

	cmd := &CommitCommand{}

	t.Run("Basic Commit", func(t *testing.T) {
		createAndAdd("test1.txt", "hello")

		res, err := cmd.Execute(context.Background(), s, []string{"commit", "-m", "first commit"})
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if !strings.Contains(res, "Commit created") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		c, _ := repo.CommitObject(head.Hash())
		if c.Message != "first commit" {
			t.Errorf("Expected message 'first commit', got '%s'", c.Message)
		}
	})

	t.Run("Commit --amend", func(t *testing.T) {
		createAndAdd("test2.txt", "world")
		_, _ = cmd.Execute(context.Background(), s, []string{"commit", "-m", "second commit"})

		// Amend
		createAndAdd("test3.txt", "amended")
		res, err := cmd.Execute(context.Background(), s, []string{"commit", "--amend", "-m", "amended commit"})
		if err != nil {
			t.Fatalf("Amend failed: %v", err)
		}

		if !strings.Contains(res, "Commit amended") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		c, _ := repo.CommitObject(head.Hash())
		if c.Message != "amended commit" {
			t.Errorf("Expected message 'amended commit', got '%s'", c.Message)
		}

		// check if text3 is there
		if _, err := c.File("test3.txt"); err != nil {
			t.Error("test3.txt not found in amended commit")
		}
	})

	t.Run("Commit Empty", func(t *testing.T) {
		_, err := cmd.Execute(context.Background(), s, []string{"commit", "-m", "empty fail"})
		if err == nil {
			t.Error("Expected error for empty commit without flag")
		}

		res, err := cmd.Execute(context.Background(), s, []string{"commit", "--allow-empty", "-m", "empty ok"})
		if err != nil {
			t.Fatalf("Empty commit failed: %v", err)
		}
		if !strings.Contains(res, "Commit created") {
			t.Errorf("Unexpected output: %s", res)
		}
	})
}
