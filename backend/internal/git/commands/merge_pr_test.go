package commands

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestMergePRCommand(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// Create Remote Repo with correct structure (Base and Feature branch)
	remotePath := filepath.Join(tempDir, "remote")
	repo, _ := gogit.PlainInit(remotePath, false)
	w, _ := repo.Worktree()

	// Base Branch (master)
	w.Filesystem.Create("main.txt")
	w.Add("main.txt")
	w.Commit("Main commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Create feature branch
	w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	})
	f, _ := w.Filesystem.Create("feature.txt")
	f.Write([]byte("Feature code"))
	f.Close()
	w.Add("feature.txt")
	w.Commit("Feature commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// Switch back to master to simulate PR target state
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")})

	// Ingest
	sm.IngestRemote(context.Background(), "origin", remotePath, 0)

	// FIX: Ensure 'feature' branch exists in SharedRemote (IngestRemote clones so it might only have remote-tracking feature)
	// We need refs/heads/feature to be present for MergePR locally on server.
	sharedRepo := sm.SharedRemotes["origin"]

	ref, err := sharedRepo.Reference(plumbing.ReferenceName("refs/remotes/origin/feature"), true)
	if err == nil {
		newRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/feature"), ref.Hash())
		_ = sharedRepo.Storer.SetReference(newRef)
	}

	// Create PR in SessionManager
	pr, _ := sm.CreatePullRequest("Feat", "Desc", "feature", "master", "Dev")

	// Execute Merge
	session, _ := sm.CreateSession("test-session")
	cmd := &MergePRCommand{}
	ctx := context.Background()

	output, err := cmd.Execute(ctx, session, []string{"merge-pr", strconv.Itoa(pr.ID), "origin"})
	if err != nil {
		t.Fatalf("MergePR failed: %v", err)
	}
	t.Log(output)

	// Verify PR status
	if pr.State != "MERGED" {
		t.Errorf("PR state is %s, expected MERGED", pr.State)
	}

	// Verify Commit on Remote 'master'
	sm.RLock()
	sharedRepo = sm.SharedRemotes["origin"]
	sm.RUnlock()

	// Must fetch refs again from storage
	refs, _ := sharedRepo.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		t.Logf("Ref: %s %s", ref.Name(), ref.Hash())
		return nil
	})

	mainRef, err := sharedRepo.Reference(plumbing.ReferenceName("refs/heads/master"), true)
	if err != nil {
		t.Fatal("Could not resolve master branch on remote")
	}

	mergeCommit, err := sharedRepo.CommitObject(mainRef.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if len(mergeCommit.ParentHashes) != 2 {
		t.Errorf("Expected 2 parents for merge commit, got %d", len(mergeCommit.ParentHashes))
	}
}
