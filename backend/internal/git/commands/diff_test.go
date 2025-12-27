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

func TestDiffStandardized(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-diff")

	// Init
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Commit 1
	f, _ := w.Filesystem.Create("file.txt")
	f.Write([]byte("foo\n"))
	f.Close()
	w.Add(".")
	w.Commit("commit 1", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Me", When: time.Now()},
	})

	// Commit 2
	f, _ = w.Filesystem.Create("file.txt")
	f.Write([]byte("foo\nbar\n"))
	f.Close()
	w.Add(".")
	w.Commit("commit 2", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Me", When: time.Now()},
	})

	cmd := &DiffCommand{}

	t.Run("Diff HEAD~1 HEAD", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"diff", "HEAD~1", "HEAD"})
		if err != nil {
			t.Fatalf("Diff failed: %v", err)
		}
		if !strings.Contains(res, "+bar") {
			t.Errorf("Expected +bar in diff, got: %s", res)
		}
	})

	t.Run("Diff Missing Args", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"diff"})
		if err == nil {
			t.Errorf("Expected error for missing args, got nil. Output: %s", res)
		}
		if !strings.Contains(err.Error(), "usage:") {
			t.Errorf("Expected usage message in error, got: %v", err)
		}
	})
}
