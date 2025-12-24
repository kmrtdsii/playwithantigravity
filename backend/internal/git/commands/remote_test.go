package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestRemoteCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-remote")
	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"}) // creates session repo

	cmd := &RemoteCommand{}

	t.Run("Add and List", func(t *testing.T) {
		_, err := cmd.Execute(context.Background(), s, []string{"remote", "add", "origin", "https://example.com/repo.git"})
		if err != nil {
			t.Fatalf("Remote add failed: %v", err)
		}

		res, err := cmd.Execute(context.Background(), s, []string{"remote", "-v"})
		if err != nil {
			t.Fatalf("Remote list failed: %v", err)
		}
		if !strings.Contains(res, "origin") || !strings.Contains(res, "https://example.com/repo.git") {
			t.Errorf("Unexpected remote list output: %s", res)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		_, err := cmd.Execute(context.Background(), s, []string{"remote", "remove", "origin"})
		if err != nil {
			t.Fatalf("Remote remove failed: %v", err)
		}

		res, err := cmd.Execute(context.Background(), s, []string{"remote"})
		if err != nil {
			t.Fatalf("Remote list failed: %v", err)
		}
		if strings.Contains(res, "origin") {
			t.Error("Origin should be removed")
		}
	})
}

func TestFetchCommand(t *testing.T) {
	// Setup: Session with a local repo and a simulated remote
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-fetch")

	// Create "Remote" repo in memory manually
	remoteSt := memory.NewStorage()
	fs := memfs.New()
	remoteRepo, _ := gogit.Init(remoteSt, fs)
	w, _ := remoteRepo.Worktree()

	// Create a dummy file to commit
	f, _ := fs.Create("dummy")
	f.Close()
	w.Add("dummy")

	_, err := w.Commit("Remote Commit", &gogit.CommitOptions{Author: &object.Signature{Name: "Remote", When: time.Now()}})
	if err != nil {
		t.Fatalf("Remote setup commit failed: %v", err)
	}

	// Register as shared remote
	sm.SharedRemotes["test-shared"] = remoteRepo

	// Local repo
	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"})
	repo := s.GetRepo()

	// Add remote
	repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"test-shared"}, // Maps to SharedRemotes key
	})

	cmd := &FetchCommand{}

	t.Run("Fetch basic", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"fetch", "origin"})
		if err != nil {
			t.Fatalf("Fetch failed: %v", err)
		}
		if !strings.Contains(res, "new branch") && !strings.Contains(res, "test-shared") {
			// output depends on implementation details
		}

		// Check if remote ref exists
		refs, _ := repo.References()
		found := false
		refs.ForEach(func(r *plumbing.Reference) error {
			if strings.Contains(r.Name().String(), "refs/remotes/origin") {
				found = true
			}
			return nil
		})
		if !found {
			t.Error("Remote refs not found after fetch")
		}
	})
}
