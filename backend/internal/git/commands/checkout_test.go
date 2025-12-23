package commands

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	git "github.com/kurobon/gitgym/backend/internal/git"
)

func TestCheckoutCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-checkout")

	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"})

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Need a commit to checkout from
	f, _ := w.Filesystem.Create("file.txt")
	f.Write([]byte("base"))
	f.Close()
	w.Add("file.txt")
	w.Commit("base commit", &gogit.CommitOptions{Author: &object.Signature{Name: "Me", When: time.Now()}})

	cmd := &CheckoutCommand{}

	t.Run("Checkout -b", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"checkout", "-b", "feature"})
		if err != nil {
			t.Fatalf("Checkout -b failed: %v", err)
		}
		if !strings.Contains(res, "Switched to a new branch 'feature'") {
			t.Errorf("Unexpected output: %s", res)
		}

		head, _ := repo.Head()
		if head.Name().Short() != "feature" {
			t.Errorf("Expected branch 'feature', got '%s'", head.Name().Short())
		}
	})

	t.Run("Checkout switch back", func(t *testing.T) {
		// Determine default branch name
		refs, _ := repo.References()
		var defaultBranch string
		refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsBranch() && r.Name().Short() != "feature" {
				defaultBranch = r.Name().Short()
			}
			return nil
		})
		if defaultBranch == "" {
			defaultBranch = "master" // Fallback fallback
		}

		res, err := cmd.Execute(context.Background(), s, []string{"checkout", defaultBranch})
		if err != nil {
			t.Fatalf("Checkout %s failed: %v", defaultBranch, err)
		}
		if !strings.Contains(res, fmt.Sprintf("Switched to branch '%s'", defaultBranch)) {
			t.Errorf("Unexpected output: %s", res)
		}
	})

	// Test brittle case (if we want to support it, or confirm it fails now)
	// git checkout -b feature2 (works)
	// git checkout -f master (fails currently because args[1] must be -b or branch?)
}
