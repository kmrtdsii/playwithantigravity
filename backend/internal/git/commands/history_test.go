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

func TestLogCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-log")

	// Init manually
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"
	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Commits
	_, err := w.Commit("first", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "User", When: time.Now()},
		AllowEmptyCommits: true,
	})
	if err != nil {
		t.Fatalf("Commit 1 failed: %v", err)
	}
	_, err = w.Commit("second", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "User", When: time.Now()},
		AllowEmptyCommits: true,
	})
	if err != nil {
		t.Fatalf("Commit 2 failed: %v", err)
	}

	cmd := &LogCommand{}

	t.Run("Log default", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"log"})
		if err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		if !strings.Contains(res, "first") || !strings.Contains(res, "second") {
			t.Error("Log output missing messages")
		}
	})

	t.Run("Log oneline", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"log", "--oneline"})
		if err != nil {
			t.Fatalf("Log oneline failed: %v", err)
		}
		if len(strings.Split(strings.TrimSpace(res), "\n")) != 2 {
			t.Error("Expected 2 lines")
		}
	})
}

func TestReflogCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-reflog")

	// Init manually
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"

	s.Reflog = append(s.Reflog, git.ReflogEntry{Hash: "abc1234", Message: "checkout: moving", Timestamp: time.Now()})

	cmd := &ReflogCommand{}
	res, err := cmd.Execute(context.Background(), s, []string{"reflog"})
	if err != nil {
		t.Fatalf("Reflog failed: %v", err)
	}
	if !strings.Contains(res, "checkout: moving") {
		t.Error("Reflog output missing entry")
	}
}

func TestDiffCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-diff")

	// Init manually
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"
	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Create common ancestor
	f, _ := w.Filesystem.Create("file.txt")
	f.Write([]byte("base\n"))
	f.Close()
	w.Add("file.txt")
	c1, _ := w.Commit("base", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Create divergence
	// Branch A
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/branchA", Create: true})
	f, _ = w.Filesystem.OpenFile("file.txt", 0x2|0x40, 0644) // truncated? or append?
	f.Write([]byte("base\nchangeA\n"))
	f.Close()
	w.Add("file.txt")
	c2, _ := w.Commit("changeA", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Reset to base and branch B ?
	// Or just diff c1 c2

	cmd := &DiffCommand{}
	res, err := cmd.Execute(context.Background(), s, []string{"diff", c1.String(), c2.String()})
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if !strings.Contains(res, "+changeA") {
		t.Errorf("Diff missing change: %s", res)
	}
}
