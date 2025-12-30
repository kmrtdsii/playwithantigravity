package commands

import (
	"context"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClone_CustomDefaultBranch verifies that cloning a repo with a default branch
// other than 'main' or 'master' (e.g. 'trunk') works correctly.
func TestClone_CustomDefaultBranch(t *testing.T) {
	sm := git.NewSessionManager()
	s, err := sm.CreateSession("test-clone-custom")
	require.NoError(t, err)

	cmd := &CloneCommand{}
	url := "https://github.com/example/trunk-repo.git"

	// 1. Setup Mock Remote Repo with 'trunk' as HEAD
	remoteRepo, err := gogit.Init(memory.NewStorage(), nil)
	require.NoError(t, err)

	rStorer := remoteRepo.Storer

	// Create commit
	blobEncoded := rStorer.NewEncodedObject()
	blobEncoded.SetType(plumbing.BlobObject)
	w, _ := blobEncoded.Writer()
	w.Write([]byte("content"))
	w.Close()
	blobHash, err := rStorer.SetEncodedObject(blobEncoded)
	require.NoError(t, err)

	treeEntry := object.TreeEntry{Name: "README.md", Mode: 0100644, Hash: blobHash}
	tree := object.Tree{Entries: []object.TreeEntry{treeEntry}}
	treeEncoded := rStorer.NewEncodedObject()
	tree.Encode(treeEncoded)
	treeHash, err := rStorer.SetEncodedObject(treeEncoded)
	require.NoError(t, err)

	commit := object.Commit{
		Author:    object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		Committer: object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		Message:   "Initial commit",
		TreeHash:  treeHash,
	}
	commitEncoded := rStorer.NewEncodedObject()
	commit.Encode(commitEncoded)
	commitHash, err := rStorer.SetEncodedObject(commitEncoded)
	require.NoError(t, err)

	// Set HEAD to refs/heads/trunk
	trunkRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/heads/trunk"), commitHash)
	err = rStorer.SetReference(trunkRef)
	require.NoError(t, err)

	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/trunk"))
	err = rStorer.SetReference(headRef)
	require.NoError(t, err)

	// Inject into Manager
	sm.SharedRemotes[url] = remoteRepo

	// 2. Execute Clone
	_, err = cmd.Execute(context.Background(), s, []string{"clone", url})
	require.NoError(t, err)

	// 3. Verify Local Repo State
	localRepo, ok := s.Repos["trunk-repo"]
	if !ok {
		t.Fatal("Repo not cloned")
	}

	head, err := localRepo.Head()
	require.NoError(t, err, "HEAD should resolve")
	assert.Equal(t, "refs/heads/trunk", head.Name().String(), "HEAD should point to trunk")
	assert.Equal(t, commitHash.String(), head.Hash().String())

	// Verify working tree file exists
	wt, _ := localRepo.Worktree()
	_, err = wt.Filesystem.Stat("README.md")
	assert.NoError(t, err, "README.md should be checked out")
}
