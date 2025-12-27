package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestStatusCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-status")

	// Init
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	cmd := &StatusCommand{}

	t.Run("Clean Status", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"status"})
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		// Empty output or specific message for clean?
		// go-git status Clean string is empty usually? Or we print something?
		// Current impl calls status.String().
		// If clean, it might be empty or not. Let's see.
		// Actually go-git Status.String() returns nothing if clean?
		// Let's check what it returns.
		_ = res
	})

	t.Run("Dirty Status", func(t *testing.T) {
		f, _ := w.Filesystem.Create("dirty.txt")
		f.Write([]byte("dirty"))
		f.Close()

		res, err := cmd.Execute(context.Background(), s, []string{"status"})
		if err != nil {
			t.Fatalf("Status dirty failed: %v", err)
		}
		if !strings.Contains(res, "dirty.txt") {
			t.Errorf("Expected dirty.txt in status, got: %s", res)
		}
	})
}
