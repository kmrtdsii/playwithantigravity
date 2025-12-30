package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

// mockHybridStorer simulates HybridStorer behavior for testing.
// It implements localStorerProvider interface by providing LocalStorer() method.
type mockHybridStorer struct {
	storage.Storer
	shared storage.Storer
}

// LocalStorer returns the underlying local storage.
// This method makes mockHybridStorer satisfy the localStorerProvider interface.
func (m *mockHybridStorer) LocalStorer() storage.Storer {
	return m.Storer
}

func TestPopulateCommits_HybridStorer_SkipsObjectIteration(t *testing.T) {
	// Create a "local" repo using mockHybridStorer
	localSt := memory.NewStorage()
	sharedSt := memory.NewStorage()

	hybridSt := &mockHybridStorer{
		Storer: localSt,
		shared: sharedSt,
	}

	// Initialize repo with hybrid storer
	localRepo, err := gogit.Init(hybridSt, memfs.New())
	require.NoError(t, err)

	// Build graph state with showAll=true
	state := &GraphState{
		Branches:       make(map[string]string),
		RemoteBranches: make(map[string]string),
		Tags:           make(map[string]string),
		References:     make(map[string]string),
	}

	// This should NOT panic and should use BFS instead of object iteration
	// Since local has no refs or commits, we expect no commits
	populateCommits(localRepo, state, true)

	assert.Empty(t, state.Commits, "HybridStorer with showAll=true should not iterate shared objects")
}

func TestPopulateCommits_NonHybrid_UsesObjectIteration(t *testing.T) {
	// Create a normal (non-hybrid) repo
	tmpDir, err := os.MkdirTemp("", "test-graph-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fs := osfs.New(tmpDir)
	err = fs.MkdirAll(".git", 0755)
	require.NoError(t, err)
	dotGit, err := fs.Chroot(".git")
	require.NoError(t, err)
	st := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	repo, err := gogit.Init(st, fs)
	require.NoError(t, err)

	// Create a commit
	wt, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)

	_, err = wt.Add("test.txt")
	require.NoError(t, err)

	_, err = wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)

	// Build graph state with showAll=true
	state := &GraphState{
		Branches:       make(map[string]string),
		RemoteBranches: make(map[string]string),
		Tags:           make(map[string]string),
		References:     make(map[string]string),
	}

	populateCommits(repo, state, true)

	// Non-hybrid repo should show the commit via object iteration
	assert.Len(t, state.Commits, 1, "Non-hybrid repo with showAll=true should iterate all objects")
}

func TestPopulateCommits_BFSFromRefs(t *testing.T) {
	// Create a normal repo with multiple commits
	tmpDir, err := os.MkdirTemp("", "test-bfs-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fs := osfs.New(tmpDir)
	err = fs.MkdirAll(".git", 0755)
	require.NoError(t, err)
	dotGit, err := fs.Chroot(".git")
	require.NoError(t, err)
	st := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	repo, err := gogit.Init(st, fs)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create first commit
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("v1"), 0644)
	require.NoError(t, err)
	_, err = wt.Add("test.txt")
	require.NoError(t, err)
	_, err = wt.Commit("commit 1", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)

	// Create second commit
	err = os.WriteFile(testFile, []byte("v2"), 0644)
	require.NoError(t, err)
	_, err = wt.Add("test.txt")
	require.NoError(t, err)
	_, err = wt.Commit("commit 2", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)

	// Build graph state with showAll=false (BFS mode)
	state := &GraphState{
		Branches:       make(map[string]string),
		RemoteBranches: make(map[string]string),
		Tags:           make(map[string]string),
		References:     make(map[string]string),
	}

	populateCommits(repo, state, false)

	// Should find both commits via BFS from HEAD
	assert.Len(t, state.Commits, 2, "BFS should find all reachable commits")
}

func TestPopulateCommits_HybridStorer_BFSStillWorks(t *testing.T) {
	// Create a "local" repo using mockHybridStorer WITH a commit
	tmpDir, err := os.MkdirTemp("", "test-hybrid-bfs-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fs := osfs.New(tmpDir)
	err = fs.MkdirAll(".git", 0755)
	require.NoError(t, err)
	dotGit, err := fs.Chroot(".git")
	require.NoError(t, err)
	localSt := filesystem.NewStorage(dotGit, cache.NewObjectLRUDefault())

	sharedSt := memory.NewStorage()
	hybridSt := &mockHybridStorer{
		Storer: localSt,
		shared: sharedSt,
	}

	repo, err := gogit.Init(hybridSt, fs)
	require.NoError(t, err)

	// Create a commit
	wt, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello"), 0644)
	require.NoError(t, err)
	_, err = wt.Add("test.txt")
	require.NoError(t, err)
	_, err = wt.Commit("test commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)

	// Build graph state with showAll=true (but should still use BFS for hybrid)
	state := &GraphState{
		Branches:       make(map[string]string),
		RemoteBranches: make(map[string]string),
		Tags:           make(map[string]string),
		References:     make(map[string]string),
	}

	populateCommits(repo, state, true)

	// Even with showAll=true, HybridStorer should use BFS and find the local commit
	assert.Len(t, state.Commits, 1, "HybridStorer with showAll=true should still find local commits via BFS")
}
