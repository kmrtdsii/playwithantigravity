package commands

import (
	"context"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestSwitchCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-switch")

	// Init
	s.InitRepo("testrepo")
	s.CurrentDir = "/testrepo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Commit init
	f, _ := w.Filesystem.Create("test.txt")
	f.Write([]byte("init"))
	f.Close()
	w.Add(".")
	w.Commit("initial", &gogit.CommitOptions{Author: git.GetDefaultSignature()})

	cmd := &SwitchCommand{}

	t.Run("Switch Create", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"switch", "-c", "feature"})
		if err != nil {
			t.Fatalf("Switch create failed: %v", err)
		}
		if !strings.Contains(res, "Switched to a new branch 'feature'") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		if head.Name().Short() != "feature" {
			t.Errorf("HEAD not at feature, got %s", head.Name().Short())
		}
	})

	t.Run("Switch Existing", func(t *testing.T) {
		// Switch back to master (default)
		// Assuming default was master or main. Let's create 'dev' manually first to switch TO.

		// Create another branch "dev"
		w.Checkout(&gogit.CheckoutOptions{
			Create: true,
			Branch: plumbing.ReferenceName("refs/heads/dev"),
		})

		// Now test switching to it using our command
		// First ensure we are NOT on dev (we are on feature from prev test? Session is shared? Yes same session variable)

		res, err := cmd.Execute(context.Background(), s, []string{"switch", "dev"})
		if err != nil {
			t.Fatalf("Switch existing failed: %v", err)
		}
		if !strings.Contains(res, "Switched to branch 'dev'") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		if head.Name().Short() != "dev" {
			t.Errorf("HEAD not at dev, got %s", head.Name().Short())
		}
	})

	t.Run("Switch Fail Missing", func(t *testing.T) {
		_, err := cmd.Execute(context.Background(), s, []string{"switch", "missing-branch"})
		if err == nil {
			t.Error("Expected error for missing branch")
		}
	})
}
