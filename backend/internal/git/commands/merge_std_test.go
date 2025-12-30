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
	"github.com/stretchr/testify/require"
)

func TestMergeCommandStandardized(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit (master)
	fs.Create("base.txt")
	w.Add("base.txt")
	_, _ = w.Commit("Base commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 2. Create Feature Branch
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/feature"), Create: true})
	fs.Create("feature.txt")
	w.Add("feature.txt")
	featureHash, _ := w.Commit("Feature commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 3. Switch back to master and diverge (for true merge)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.Master, Force: true})
	fs.Create("master.txt")
	w.Add("master.txt")
	masterHash, _ := w.Commit("Master commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}

	cmd := &MergeCommand{}

	// Test 1: Dry Run
	output, err := cmd.Execute(context.Background(), session, []string{"merge", "--dry-run", "feature"})
	assert.NoError(t, err)
	assert.Contains(t, output, "[dry-run]")
	assert.Contains(t, output, "Would create merge commit")

	// Verify NO merge happened
	head, _ := r.Head()
	assert.Equal(t, masterHash, head.Hash())

	// Test 2: Actual Merge
	output, err = cmd.Execute(context.Background(), session, []string{"merge", "feature"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Merge made by the 'ort' strategy")

	// Verify Merge Commit
	head, headErr := r.Head()
	require.NoError(t, headErr)
	require.NotNil(t, head)

	assert.NotEqual(t, masterHash, head.Hash())
	cObj, cErr := r.CommitObject(head.Hash())
	require.NoError(t, cErr)
	require.NotNil(t, cObj)

	assert.Equal(t, 2, cObj.NumParents())
	if cObj.NumParents() >= 2 {
		p1, _ := cObj.Parent(0)
		p2, _ := cObj.Parent(1)
		// assert.Equal is fine here as we established safety
		assert.Equal(t, masterHash, p1.Hash)
		assert.Equal(t, featureHash, p2.Hash)
	}

	// Verify Content
	_, err = fs.Stat("base.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("master.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("feature.txt")
	assert.NoError(t, err)
}

func TestMergeFastForward(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial
	fs.Create("base.txt")
	w.Add("base.txt")
	_, _ = w.Commit("Base", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", When: time.Now()},
	})

	// 2. Feature ahead of master
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/feature"), Create: true})
	fs.Create("feature.txt")
	w.Add("feature.txt")
	featureHash, _ := w.Commit("Feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", When: time.Now()},
	})

	// 3. Switch to master (behind)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.Master, Force: true})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &MergeCommand{}

	// Test FF
	output, err := cmd.Execute(context.Background(), session, []string{"merge", "feature"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Fast-forward")

	// Verify HEAD moved to feature
	head, _ := r.Head()
	assert.Equal(t, featureHash, head.Hash())
}

// TestMergeEmptyTreeCommits tests merging branches where both have empty commits
// (no actual file changes). This was a bug where the merge failed with
// "cannot create empty commit: clean working tree".
func TestMergeEmptyTreeCommits(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial empty commit (base)
	baseHash, _ := w.Commit("Base (empty)", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		AllowEmptyCommits: true,
	})

	// 2. Create feature branch with empty commit
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/feature"), Create: true})
	featureHash, _ := w.Commit("Feature (empty)", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		AllowEmptyCommits: true,
	})

	// 3. Switch back to master and make empty commit (diverge)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.Master, Force: true})
	masterHash, _ := w.Commit("Master (empty)", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
		AllowEmptyCommits: true,
	})

	// Verify we have diverged
	require.NotEqual(t, baseHash, masterHash)
	require.NotEqual(t, baseHash, featureHash)
	require.NotEqual(t, masterHash, featureHash)

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &MergeCommand{}

	// Test: Merge should succeed even with empty trees
	output, err := cmd.Execute(context.Background(), session, []string{"merge", "feature"})
	require.NoError(t, err, "Merge of empty tree branches should not fail")
	assert.Contains(t, output, "Merge made by the 'ort' strategy")

	// Verify merge commit was created
	head, _ := r.Head()
	require.NotEqual(t, masterHash, head.Hash(), "HEAD should have moved to merge commit")

	mergeCommit, _ := r.CommitObject(head.Hash())
	require.NotNil(t, mergeCommit)
	assert.Equal(t, 2, mergeCommit.NumParents(), "Merge commit should have 2 parents")
}
