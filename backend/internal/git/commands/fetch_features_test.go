package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func setupTestRepo(t *testing.T) (*git.SessionManager, *git.Session, *gogit.Repository) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// Create Origin Remote
	originPath := filepath.Join(tempDir, "remote")
	r, _ := gogit.PlainInit(originPath, false)
	w, _ := r.Worktree()

	// Commit 1: main
	w.Filesystem.Create("main.txt")
	w.Add("main.txt")
	w.Commit("Main Commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Branch: feature
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature"), Create: true})
	w.Filesystem.Create("feature.txt")
	w.Add("feature.txt")
	w.Commit("Feature Commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Tag: v1.0
	headRef, _ := r.Head()
	r.CreateTag("v1.0", headRef.Hash(), &gogit.CreateTagOptions{
		Message: "Release v1.0",
		Tagger:  &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Switch back to main
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")}) // go-git defaults master

	// Create Session & Ingest
	sm.IngestRemote(context.Background(), "origin", originPath)
	session, _ := sm.CreateSession("test-session")

	// Clone
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	repo := session.GetRepo()
	return sm, session, repo
}

func TestFetch_SpecificBranch(t *testing.T) {
	_, session, repo := setupTestRepo(t)
	fetchCmd := &FetchCommand{}

	// Act: fetch origin feature
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "origin", "feature"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "feature -> origin/feature")

	// Verify ref exists
	refName := plumbing.ReferenceName("refs/remotes/origin/feature")
	_, err = repo.Reference(refName, true)
	assert.NoError(t, err)
}

func TestFetch_Tags(t *testing.T) {
	_, session, repo := setupTestRepo(t)
	fetchCmd := &FetchCommand{}

	// Pre-check: Tag shouldn't exist locally yet (Clone implies fetch, but maybe tags weren't default? Go-git clone fetches tags by default usually. Let's delete it first to be sure)
	repo.DeleteTag("v1.0")

	// Act: fetch --tags
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "--tags"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "[new tag]")
	assert.Contains(t, output, "v1.0")

	// Verify tag exists
	_, err = repo.Tag("v1.0")
	assert.NoError(t, err)
}

func TestFetch_Prune(t *testing.T) {
	// 1. Setup
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// Create Remote
	originPath := filepath.Join(tempDir, "remote")
	r, _ := gogit.PlainInit(originPath, false)
	w, _ := r.Worktree()

	// Commit 1: main
	w.Filesystem.Create("main.txt")
	w.Add("main.txt")
	w.Commit("Main Commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Branch: to-be-deleted
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("tbd"), Create: true})
	w.Commit("TBD", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Ingest
	sm.IngestRemote(context.Background(), "origin", originPath)
	session, _ := sm.CreateSession("test-session")

	// Clone (Gets everything)
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	repo := session.GetRepo()
	_, err := repo.Reference("refs/remotes/origin/tbd", true)
	assert.NoError(t, err, "tbd branch should exist initially")

	// 2. Delete branch on remote
	// We need to modify the "remote" repo directly.
	// Note: In our simulation, the "remote" IS the repo in IngestRemote path or SharedRemotes.
	// Since we used IngestRemote, it points to local disk.
	err = r.Storer.RemoveReference(plumbing.ReferenceName("refs/heads/tbd"))
	assert.NoError(t, err)

	// 3. Fetch -p
	fetchCmd := &FetchCommand{}
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "-p"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "[deleted]")
	assert.Contains(t, output, "tbd")

	_, err = repo.Reference("refs/remotes/origin/tbd", true)
	assert.Error(t, err, "remote branch ref should be gone")
}

func TestFetch_DryRun(t *testing.T) {
	// _, session, repo := setupTestRepo(t)
	fetchCmd := &FetchCommand{}

	// Create a NEW branch on remote that wasn't cloned
	// We need access to the remote repo. setupTestRepo doesn't return it directly but returns session.
	// But in this test setup, we used a local path for remote "remote".
	// We can reconstruct the remote path or modify setupTestRepo, OR just add it to the 'origin' which is mapped to a local path.
	// Getting the path from session manager's ingest might be hard.
	// Let's just create a new test setup inline or assume the path.
	tempDir := t.TempDir() // Wait, t.TempDir is unique per test? No, setupTestRepo creates a new one.
	// We can't access `originPath` from here easily without returning it.

	// Better approach: Let's assume the "remote" is accessible via the session manager if we peek,
	// or we can just fetch a branch that doesn't exist? No that would error.

	// Let's modify setupTestRepo to return the remote repo object or path?
	// Or easier: Just adding `feature-2` branch to the repo at `originPath`.
	// But `setupTestRepo` hides `originPath`.

	// Let's just manually create the scenario here since it's cleaner.
	sm := git.NewSessionManager()
	sm.DataDir = filepath.Join(tempDir, "data")
	originPath := filepath.Join(tempDir, "remote_dryrun")

	r, _ := gogit.PlainInit(originPath, false)
	w, _ := r.Worktree()
	w.Filesystem.Create("main.txt")
	w.Add("main.txt")
	w.Commit("Init", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

	sm.IngestRemote(context.Background(), "origin", originPath)
	session, _ := sm.CreateSession("test-dryrun")

	// Clone
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	// Now create a new branch on remote
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("new-feature"), Create: true})
	w.Commit("New Feature", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

	repo := session.GetRepo()

	// Act: fetch origin new-feature -n
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "-n", "origin", "new-feature"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "[dry-run]")
	assert.Contains(t, output, "new-feature")

	// Verify ref does NOT exist
	refName := plumbing.ReferenceName("refs/remotes/origin/new-feature")
	_, err = repo.Reference(refName, true)
	assert.Error(t, err, "Ref should not exist after dry-run")
}
