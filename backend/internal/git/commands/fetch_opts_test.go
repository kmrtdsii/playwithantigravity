package commands

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestFetchCommand_Tags(t *testing.T) {
	// 1. Setup
	sm := git.NewSessionManager()
	sm.DataDir = t.TempDir()

	// In-Memory Remote
	r := memory.NewStorage()
	fs := memfs.New()
	originRepo, _ := gogit.Init(r, fs)
	originURL := "https://example.com/origin-tags.git"

	w, _ := originRepo.Worktree()
	fs.Create("README.md")
	w.Add("README.md")
	// Add Tag on Remote (using hash directly)
	initHash, err := w.Commit("Init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = originRepo.CreateTag("v1.0.0", initHash, &gogit.CreateTagOptions{
		Tagger:  &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
		Message: "Release v1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	sm.Lock()
	sm.SharedRemotes[originURL] = originRepo
	sm.Unlock()

	session, _ := sm.CreateSession("test-tags")
	cloneCmd := &CloneCommand{}
	_, err = cloneCmd.Execute(context.Background(), session, []string{"clone", originURL})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Add v2.0.0 to remote (Simulate new tag appearing)
	w.Commit("Update", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})
	headRef2, _ := originRepo.Head()
	_, err = originRepo.CreateTag("v2.0.0", headRef2.Hash(), nil) // Lightweight tag
	if err != nil {
		t.Fatal(err)
	}

	// 3. Fetch with --tags
	fetchCmd := &FetchCommand{}
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "--tags", "origin"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Fetch tags output: %s", output)

	// 4. Verify Local Tags
	localRepo := session.GetRepo()
	_, err = localRepo.Reference("refs/tags/v2.0.0", true)
	if err != nil {
		t.Errorf("Failed to fetch tag v2.0.0: %v", err)
	}
}

func TestFetchCommand_Prune(t *testing.T) {
	// 1. Setup
	sm := git.NewSessionManager()
	sm.DataDir = t.TempDir()

	r := memory.NewStorage()
	fs := memfs.New()
	originRepo, _ := gogit.Init(r, fs)
	originURL := "https://example.com/origin-prune.git"

	w, _ := originRepo.Worktree()
	fs.Create("README.md")
	w.Add("README.md")
	// Create a branch "feature" on remote
	initHash, err := w.Commit("Init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	featureRef := plumbing.NewHashReference("refs/heads/feature", initHash)
	originRepo.Storer.SetReference(featureRef)

	sm.Lock()
	sm.SharedRemotes[originURL] = originRepo
	sm.Unlock()

	session, _ := sm.CreateSession("test-prune")
	cloneCmd := &CloneCommand{}
	_, err = cloneCmd.Execute(context.Background(), session, []string{"clone", originURL})
	if err != nil {
		t.Fatal(err)
	}

	// Verify we have origin/feature locally
	localRepo := session.GetRepo()
	_, errPrune := localRepo.Reference("refs/remotes/origin/feature", true)
	if errPrune != nil {
		t.Errorf("Pre-condition failed: origin/feature missing")
	}

	// 2. Delete branch on Remote
	originRepo.Storer.RemoveReference(featureRef.Name())

	// 3. Fetch (Normal) - Should NOT prune
	fetchCmd := &FetchCommand{}
	_, err = fetchCmd.Execute(context.Background(), session, []string{"fetch", "origin"})
	if err != nil {
		t.Fatal(err)
	}
	// Verify it still exists
	_, err = localRepo.Reference("refs/remotes/origin/feature", true)
	if err != nil {
		t.Errorf("Fetch without prune should NOT delete remote branch")
	}

	// 4. Fetch with --prune
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "--prune", "origin"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Fetch prune output: %s", output)

	// Verify it is gone
	_, err = localRepo.Reference("refs/remotes/origin/feature", true)
	if err == nil {
		t.Errorf("Fetch --prune FAILED to delete remote branch")
	}
}
