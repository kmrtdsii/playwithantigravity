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
	"github.com/stretchr/testify/assert"
)

func TestCheckoutOrphan(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)

	// Create initial commit
	w, _ := r.Worktree()
	_, _ = fs.Create("README.md")
	_, _ = w.Add("README.md")
	commit, _ := w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}

	cmd := &CheckoutCommand{}

	// Test: git checkout --orphan new-root
	output, err := cmd.Execute(context.Background(), session, []string{"checkout", "--orphan", "new-root"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Switched to a new branch 'new-root' (orphan)")

	// Verify HEAD
	// for unborn branch, HEAD is symbolic. repo.Head() attempts to resolve it and might return error if target is missing.
	// We use repo.Reference(plumbing.HEAD, false) to get the symbolic ref itself.
	head, err := r.Reference(plumbing.HEAD, false)
	assert.NoError(t, err)
	assert.Equal(t, plumbing.SymbolicReference, head.Type())
	assert.Equal(t, "refs/heads/new-root", head.Target().String())

	// Verify it is unborn
	// Attempting to resolve it should fail
	_, err = r.Reference(head.Target(), true)
	assert.Error(t, err)

	// Verify working directory is preserved (file exists)
	_, err = fs.Stat("README.md")
	assert.NoError(t, err)

	// Make a commit
	newCommitHash, err := w.Commit("Root commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	assert.NoError(t, err)

	// This commit should have NO parents because we were on orphan branch
	cObj, err := r.CommitObject(newCommitHash)
	assert.NoError(t, err)
	assert.Equal(t, 0, cObj.NumParents())
	assert.NotEqual(t, commit, newCommitHash)
}
