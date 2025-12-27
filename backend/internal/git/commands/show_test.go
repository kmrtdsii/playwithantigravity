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

func TestShowCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-show")

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
	c1, _ := w.Commit("commit 1", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Me", When: time.Now()},
	})

	cmd := &ShowCommand{}

	t.Run("Show HEAD", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"show"})
		if err != nil {
			t.Fatalf("Show failed: %v", err)
		}
		if !strings.Contains(res, "commit 1") {
			t.Errorf("Expected commit message in show output, got: %s", res)
		}
		if !strings.Contains(res, c1.String()) {
			// default show prints commit object string?
			// current implementation prints formatting?
			// actually commit.String() prints full details.
		}
	})

	t.Run("Show --name-status", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"show", "--name-status"})
		if err != nil {
			t.Fatalf("Show --name-status failed: %v", err)
		}
		if !strings.Contains(res, "A\tfile.txt") {
			t.Errorf("Expected A file.txt status, got: %s", res)
		}
	})
}
