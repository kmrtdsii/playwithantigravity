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

	_, _ = s.InitRepo("repo")
	s.CurrentDir = "/repo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Need a commit to checkout from
	f, _ := w.Filesystem.Create("file.txt")
	_, _ = f.Write([]byte("base"))
	_ = f.Close()
	_, _ = w.Add("file.txt")
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

	t.Run("Checkout Force", func(t *testing.T) {
		// Determine default branch name again or use hardcoded if known
		var defaultBranch string
		refs, _ := repo.References()
		refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsBranch() && r.Name().Short() != "feature" {
				defaultBranch = r.Name().Short()
			}
			return nil
		})
		if defaultBranch == "" {
			defaultBranch = "master"
		}

		// Create a dirty state
		f, _ := w.Filesystem.Create("file.txt")
		f.Write([]byte("dirty"))
		f.Close()

		// Attempt checkout without force (should fail or carry over? Git checkout carries over if no conflict, but if modifying same file...)
		// If we switch to 'master' (which has 'base'), it should conflict/fail.

		// However, with memfs and our simplistic implementation, let's verify -f overwrites.
		res, err := cmd.Execute(context.Background(), s, []string{"checkout", "-f", defaultBranch})
		if err != nil {
			t.Fatalf("Checkout -f failed: %v", err)
		}
		if !strings.Contains(res, fmt.Sprintf("Switched to branch '%s'", defaultBranch)) {
			t.Errorf("Unexpected output: %s", res)
		}

		// Check content reverted to master's version
		f2, _ := w.Filesystem.Open("file.txt")
		buf := make([]byte, 100)
		n, _ := f2.Read(buf)
		content := string(buf[:n])
		if content != "base" {
			t.Errorf("Expected 'base', got '%s'", content)
		}
	})
}
