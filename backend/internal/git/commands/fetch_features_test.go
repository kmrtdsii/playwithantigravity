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

// TestSetupResult holds all objects needed for fetch tests
type TestSetupResult struct {
	SM         *git.SessionManager
	Session    *git.Session
	Repo       *gogit.Repository
	RemoteRepo *gogit.Repository
	RemoteWT   *gogit.Worktree
}

func setupTestRepo(t *testing.T) *TestSetupResult {
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

	// Branch: feature (but don't add commits yet - we'll do that AFTER clone for the test)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature"), Create: true})

	// Tag: v1.0
	headRef, _ := r.Head()
	r.CreateTag("v1.0", headRef.Hash(), &gogit.CreateTagOptions{
		Message: "Release v1.0",
		Tagger:  &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Switch back to main
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")}) // go-git defaults master

	// Create Session & Ingest
	sm.IngestRemote(context.Background(), "origin", originPath, 0)
	session, _ := sm.CreateSession("test-session")

	// Clone
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	repo := session.GetRepo()
	return &TestSetupResult{
		SM:         sm,
		Session:    session,
		Repo:       repo,
		RemoteRepo: r,
		RemoteWT:   w,
	}
}

func TestFetch_SpecificBranch(t *testing.T) {
	setup := setupTestRepo(t)
	fetchCmd := &FetchCommand{}

	// AFTER clone: add a new commit on the remote's feature branch
	// We use the disk repo (RemoteWT) to create the commit, then copy objects to SharedRemotes
	setup.RemoteWT.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("feature")})
	setup.RemoteWT.Filesystem.Create("feature.txt")
	setup.RemoteWT.Add("feature.txt")
	featureCommit, _ := setup.RemoteWT.Commit("Feature Commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Copy objects from disk repo to SharedRemotes so fetch can see them
	// Get the SharedRemotes repo
	for _, sharedRepo := range setup.SM.SharedRemotes {
		// Copy the commit object
		git.CopyCommitRecursive(setup.RemoteRepo, sharedRepo, featureCommit)
		// Update the branch reference
		sharedRepo.Storer.SetReference(plumbing.NewHashReference(
			plumbing.NewBranchReferenceName("feature"),
			featureCommit,
		))
		break // Only need to update one (they may all point to same repo)
	}

	// Act: fetch origin feature
	output, err := fetchCmd.Execute(context.Background(), setup.Session, []string{"fetch", "origin", "feature"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "feature -> origin/feature")

	// Verify ref exists
	refName := plumbing.ReferenceName("refs/remotes/origin/feature")
	_, err = setup.Repo.Reference(refName, true)
	assert.NoError(t, err)
}

func TestFetch_Tags(t *testing.T) {
	setup := setupTestRepo(t)
	fetchCmd := &FetchCommand{}

	// Pre-check: Tag shouldn't exist locally yet (Clone implies fetch, but maybe tags weren't default? Go-git clone fetches tags by default usually. Let's delete it first to be sure)
	setup.Repo.DeleteTag("v1.0")

	// Act: fetch --tags
	output, err := fetchCmd.Execute(context.Background(), setup.Session, []string{"fetch", "--tags"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "[new tag]")
	assert.Contains(t, output, "v1.0")

	// Verify tag exists
	_, err = setup.Repo.Tag("v1.0")
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

	// Branch: to-be-deleted (add a file so it has content)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("tbd"), Create: true})
	w.Filesystem.Create("tbd.txt")
	w.Add("tbd.txt")
	w.Commit("TBD", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Back to master
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})

	// Ingest
	sm.IngestRemote(context.Background(), "origin", originPath, 0)
	session, _ := sm.CreateSession("test-session")

	// Clone (Gets everything)
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	repo := session.GetRepo()
	_, err := repo.Reference("refs/remotes/origin/tbd", true)
	assert.NoError(t, err, "tbd branch should exist initially")

	// 2. Delete branch on SharedRemotes repo (not disk repo)
	// Get the SharedRemotes repo that fetch will actually read from
	sharedRemote := sm.SharedRemotes[originPath]
	if sharedRemote == nil {
		t.Skip("SharedRemote not found for originPath - test setup issue")
	}
	err = sharedRemote.Storer.RemoveReference(plumbing.ReferenceName("refs/heads/tbd"))
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
	fetchCmd := &FetchCommand{}

	// Create inline setup
	tempDir := t.TempDir()
	sm := git.NewSessionManager()
	sm.DataDir = filepath.Join(tempDir, "data")
	originPath := filepath.Join(tempDir, "remote_dryrun")

	r, _ := gogit.PlainInit(originPath, false)
	w, _ := r.Worktree()
	w.Filesystem.Create("main.txt")
	w.Add("main.txt")
	w.Commit("Init", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

	sm.IngestRemote(context.Background(), "origin", originPath, 0)
	session, _ := sm.CreateSession("test-dryrun")

	// Clone
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	// Now create new-feature branch on SharedRemotes repo (not disk repo)
	sharedRemote := sm.SharedRemotes[originPath]
	if sharedRemote == nil {
		t.Skip("SharedRemote not found for originPath - test setup issue")
	}
	// Get worktree and create new branch with commit
	sharedWT, err := sharedRemote.Worktree()
	if err != nil {
		t.Skipf("Could not get SharedRemote worktree: %v", err)
	}
	sharedWT.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("new-feature"), Create: true})
	sharedWT.Filesystem.Create("new-feature.txt")
	sharedWT.Add("new-feature.txt")
	sharedWT.Commit("New Feature", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

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
