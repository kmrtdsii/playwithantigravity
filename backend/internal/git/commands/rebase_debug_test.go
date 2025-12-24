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

func TestRebaseShortHash(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit
	fs.Create("base.txt")
	w.Add("base.txt")
	baseHash, _ := w.Commit("Base commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &RebaseCommand{}

	// Short hash
	hashStr := baseHash.String()
	shortHash := hashStr[:7]

	// Try rebase using short hash as upstream
	output, err := cmd.Execute(context.Background(), session, []string{"rebase", shortHash})
	// Now this should SUCCEED
	assert.NoError(t, err)
	assert.Contains(t, output, "up to date")
}

func TestRebaseDisjoint(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Branch A
	fs.Create("a.txt")
	w.Add("a.txt")
	aHash, _ := w.Commit("A commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 2. Orphan Branch B (disjoint)
	// Manual orphan setup
	orphanRefName := plumbing.ReferenceName("refs/heads/orphan")
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, orphanRefName)
	r.Storer.SetReference(headRef)

	fs.Create("b.txt")
	w.Add("b.txt")
	_, _ = w.Commit("B commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &RebaseCommand{}

	// Ensure we are on orphan branch
	_ = w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/orphan")})

	// git rebase <upstream> -> fails without --root
	_, err := cmd.Execute(context.Background(), session, []string{"rebase", aHash.String()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Use --root")

	// Try with --root --onto
	// git rebase --root --onto <aHash>
	// This should replay B commit onto A commit.
	output, err := cmd.Execute(context.Background(), session, []string{"rebase", "--root", "--onto", aHash.String()})
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully rebased")

	// Verify HEAD parent is A
	headP, _ := r.Head()
	cObj, _ := r.CommitObject(headP.Hash())
	parent, _ := cObj.Parent(0)
	assert.Equal(t, aHash, parent.Hash)

	// Verify content: has a.txt (from base) and b.txt (from replay)
	_, err = fs.Stat("a.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("b.txt")
	assert.NoError(t, err)
}
