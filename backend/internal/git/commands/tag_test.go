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

func TestTagCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-tag")

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
	_ = c1 // unused

	cmd := &TagCommand{}

	t.Run("Create Lightweight Tag", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"tag", "v1.0"})
		if err != nil {
			t.Fatalf("Create tag failed: %v", err)
		}
		if !strings.Contains(res, "Created tag v1.0") {
			t.Errorf("Unexpected output: %s", res)
		}

		res, err = cmd.Execute(context.Background(), s, []string{"tag"})
		if err != nil {
			t.Fatalf("List tags failed: %v", err)
		}
		if !strings.Contains(res, "v1.0") {
			t.Errorf("Tag v1.0 not found in list")
		}
	})

	t.Run("Create Annotated Tag", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"tag", "-a", "v2.0", "-m", "release 2.0"})
		if err != nil {
			t.Fatalf("Create annotated tag failed: %v", err)
		}
		if !strings.Contains(res, "Created annotated tag v2.0") {
			t.Errorf("Unexpected output: %s", res)
		}

		res, err = cmd.Execute(context.Background(), s, []string{"tag"})
		if !strings.Contains(res, "v2.0") {
			t.Errorf("Tag v2.0 not found in list")
		}
	})

	t.Run("Delete Tag", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"tag", "-d", "v1.0"})
		if err != nil {
			t.Fatalf("Delete tag failed: %v", err)
		}
		if !strings.Contains(res, "Deleted tag v1.0") {
			t.Errorf("Unexpected output: %s", res)
		}

		res, err = cmd.Execute(context.Background(), s, []string{"tag"})
		if strings.Contains(res, "v1.0") {
			t.Errorf("Tag v1.0 still found in list")
		}
	})
}
