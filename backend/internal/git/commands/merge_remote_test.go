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

func TestMerge_RemoteBranch(t *testing.T) {
	// 1. Setup with Remote
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// Create Origin Remote
	originPath := filepath.Join(tempDir, "remote")
	r, _ := gogit.PlainInit(originPath, false)
	w, _ := r.Worktree()

	// Ensure we are on 'main'
	_ = w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/main"),
		Force:  true,
		Create: true,
	})

	// Commit 1 (Base)
	w.Filesystem.Create("base.txt")
	w.Add("base.txt")
	w.Commit("Base", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

	// Commit 2 (Remote Ahead)
	w.Filesystem.Create("remote.txt")
	w.Add("remote.txt")
	w.Commit("Remote Ahead", &gogit.CommitOptions{Author: &object.Signature{Name: "Dev", Email: "d", When: time.Now()}})

	sm.IngestRemote(context.Background(), "origin", originPath)
	session, _ := sm.CreateSession("test-merge-remote")

	// Clone (Gets everything, origin/main matches remote main)
	cloneCmd := &CloneCommand{}
	cloneCmd.Execute(context.Background(), session, []string{"clone", originPath})

	repo := session.GetRepo()

	// DEBUG: Verify origin/main points to tip (Commit 2)
	head, _ := repo.Head()
	parentHash := head.Hash() // Commit 2

	remRef, err := repo.Reference(plumbing.ReferenceName("refs/remotes/origin/main"), true)
	assert.NoError(t, err)
	assert.Equal(t, parentHash, remRef.Hash(), "Clone should update origin/main to tip")

	// Reset local main to Base
	// Current HEAD is at Commit 2.
	head, _ = repo.Head()
	parentHash = head.Hash()

	// Find base commit (parent of head)
	c, _ := repo.CommitObject(head.Hash())
	parent, _ := c.Parent(0)

	// Reset --hard to parent (Base)
	wLocal, _ := repo.Worktree()
	wLocal.Reset(&gogit.ResetOptions{Commit: parent.Hash, Mode: gogit.HardReset})

	// Verify we are behind
	currHead, _ := repo.Head()
	assert.Equal(t, parent.Hash, currHead.Hash())

	// ACT: git merge origin/main
	mergeCmd := &MergeCommand{}
	output, err := mergeCmd.Execute(context.Background(), session, []string{"merge", "origin/main"})

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "Fast-forward")

	// Verify we moved forward to Commit 2
	newHead, _ := repo.Head()
	assert.Equal(t, parentHash, newHead.Hash()) // Should match original tip
}
