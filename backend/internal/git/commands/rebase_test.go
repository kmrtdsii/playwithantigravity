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

func TestRebaseOnto(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit (Base)
	fs.Create("base.txt")
	w.Add("base.txt")
	baseHash, _ := w.Commit("Base commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 2. Upstream Branch (master)
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.Master, Force: true})
	fs.Create("upstream.txt")
	w.Add("upstream.txt")
	_, _ = w.Commit("Upstream commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 3. Create 'onto' branch (onto-target) distinct from upstream
	// Reset to base and make divergent path
	w.Checkout(&gogit.CheckoutOptions{Hash: baseHash, Force: true})
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/onto-target"), Create: true, Force: true})
	fs.Create("onto.txt")
	w.Add("onto.txt")
	ontoHash, _ := w.Commit("Onto commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// 4. Create Feature Branch on top of Upstream
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.Master, Force: true})
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/feature"), Create: true, Force: true})
	fs.Create("feature.txt")
	w.Add("feature.txt")
	featureHash, _ := w.Commit("Feature commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})

	// HEAD is now 'feature', pointing to featureHash.
	// featureHash parent is upstreamHash.
	// We want to rebase 'feature' ONTO 'onto-target', claiming 'master' is upstream (old base).

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}

	cmd := &RebaseCommand{}

	// git rebase --onto onto-target master feature
	// This should replay (master..feature] onto onto-target.
	// Result: feature branch points to new commit with parent ontoHash.
	output, err := cmd.Execute(context.Background(), session, []string{"rebase", "--onto", "onto-target", "master", "feature"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Successfully rebased")

	// Verify HEAD is on feature
	head, _ := r.Head()
	assert.Equal(t, "refs/heads/feature", head.Name().String())

	// Verify HEAD hash is new
	assert.NotEqual(t, featureHash, head.Hash())

	// Verify Parent is OntoHash
	cObj, _ := r.CommitObject(head.Hash())
	assert.Equal(t, 1, cObj.NumParents())
	parent, _ := cObj.Parent(0)
	assert.Equal(t, ontoHash, parent.Hash)

	// Verify content
	// Should have onto.txt (from new base) and feature.txt (replayed)
	_, err = fs.Stat("onto.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("feature.txt")
	assert.NoError(t, err)

	// upstream.txt should be gone or not present depending on how reset works?
	// upstream.txt was in master. rebase --onto onto-target master feature means:
	// take commits from master..feature (which is just 'Feature commit')
	// apply on onto-target.
	// onto-target has base.txt, onto.txt.
	// master had base.txt, upstream.txt.
	// 'Feature commit' added feature.txt.
	// Result should have base.txt, onto.txt, feature.txt.
	// (upstream.txt is left behind in master, not reachable from new feature).

	_, err = fs.Stat("upstream.txt")
	assert.Error(t, err, "upstream.txt should not exist in rebased branch")
}
