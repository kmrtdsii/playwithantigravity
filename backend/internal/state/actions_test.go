package state

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBareRepository(t *testing.T) {
	// Setup temp dir for GITGYM_DATA_ROOT
	tmpDir := t.TempDir()
	t.Setenv("GITGYM_DATA_ROOT", tmpDir)

	// Initialize SessionManager
	sm := &SessionManager{
		sessions:          make(map[string]*Session),
		SharedRemotes:     make(map[string]*gogit.Repository),
		SharedRemotePaths: make(map[string]string),
	}

	// Create a mock session
	sessionID := "test-session-id"
	session := &Session{
		ID:         sessionID,
		Filesystem: memfs.New(),
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
	}
	sm.sessions[sessionID] = session

	t.Run("Success", func(t *testing.T) {
		repoName := "my-new-repo"
		err := sm.CreateBareRepository(context.Background(), sessionID, repoName)
		require.NoError(t, err)

		// 1. Check if repo was registered in SharedRemotes
		assert.Contains(t, sm.SharedRemotes, repoName)
		assert.Contains(t, sm.SharedRemotePaths, repoName)

		// 2. Check if directory exists on disk
		repoPath := sm.SharedRemotePaths[repoName]
		assert.DirExists(t, repoPath)
		assert.True(t, filepath.Base(filepath.Dir(repoPath)) == "remotes")

		// CreateBareRepository only creates the remote. Ideally, clients would then clone it.
		// We do NOT check session state here as it shouldn't be modified.
	})

	t.Run("Invalid Name", func(t *testing.T) {
		err := sm.CreateBareRepository(context.Background(), sessionID, "invalid name!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repository name")
	})

	t.Run("Cleanup Existing", func(t *testing.T) {
		// Create another repo, which should remove the previous one (Single Residency)
		repoName2 := "another-repo"
		err := sm.CreateBareRepository(context.Background(), sessionID, repoName2)
		require.NoError(t, err)

		assert.Contains(t, sm.SharedRemotes, repoName2)
		// "my-new-repo" should be removed from disk?
		// Note: The map is reset in CreateBareRepository, so checking map is enough
		assert.NotContains(t, sm.SharedRemotes, "my-new-repo")
	})
}

// TestRemoveRemote tests the RemoveRemote function including PR cleanup
func TestRemoveRemote(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITGYM_DATA_ROOT", tmpDir)

	sm := &SessionManager{
		sessions:          make(map[string]*Session),
		SharedRemotes:     make(map[string]*gogit.Repository),
		SharedRemotePaths: make(map[string]string),
		PullRequests:      []*PullRequest{},
	}

	sessionID := "test-session"
	session := &Session{
		ID:         sessionID,
		Filesystem: memfs.New(),
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
	}
	sm.sessions[sessionID] = session

	t.Run("RemoveRemote clears SharedRemotes and PRs", func(t *testing.T) {
		// Setup: Create a bare repository
		err := sm.CreateBareRepository(context.Background(), sessionID, "test-repo")
		require.NoError(t, err)
		assert.Contains(t, sm.SharedRemotes, "test-repo")

		// Add some PRs
		sm.PullRequests = []*PullRequest{
			{ID: 1, Title: "PR1", State: "OPEN"},
			{ID: 2, Title: "PR2", State: "OPEN"},
		}
		require.Len(t, sm.PullRequests, 2)

		// Execute RemoveRemote
		err = sm.RemoveRemote("test-repo")
		require.NoError(t, err)

		// Verify: SharedRemotes should be empty
		assert.Empty(t, sm.SharedRemotes, "SharedRemotes should be cleared")
		assert.Empty(t, sm.SharedRemotePaths, "SharedRemotePaths should be cleared")

		// Verify: PullRequests should be empty (key behavior)
		assert.Empty(t, sm.PullRequests, "PullRequests should be cleared when remote is removed")
	})

	t.Run("RemoveRemote returns error for non-existent remote", func(t *testing.T) {
		err := sm.RemoveRemote("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestSingleResidencySpecification explicitly documents and tests the Single Residency behavior
// This is the INTENDED DESIGN: only one remote can exist at a time
func TestSingleResidencySpecification(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GITGYM_DATA_ROOT", tmpDir)

	sm := &SessionManager{
		sessions:          make(map[string]*Session),
		SharedRemotes:     make(map[string]*gogit.Repository),
		SharedRemotePaths: make(map[string]string),
		PullRequests:      []*PullRequest{},
	}

	sessionID := "test-session"
	session := &Session{
		ID:         sessionID,
		Filesystem: memfs.New(),
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
	}
	sm.sessions[sessionID] = session

	t.Run("Creating Repo B removes Repo A (Single Residency)", func(t *testing.T) {
		// Create Repo A
		err := sm.CreateBareRepository(context.Background(), sessionID, "repo-A")
		require.NoError(t, err)
		assert.Contains(t, sm.SharedRemotes, "repo-A")

		// Create Repo B
		err = sm.CreateBareRepository(context.Background(), sessionID, "repo-B")
		require.NoError(t, err)

		// SPECIFICATION: Repo A should no longer exist
		assert.NotContains(t, sm.SharedRemotes, "repo-A", "Single Residency: repo-A should be removed when repo-B is created")
		assert.Contains(t, sm.SharedRemotes, "repo-B", "repo-B should exist")

		// Only one remote should exist
		assert.Equal(t, 3, len(sm.SharedRemotes), "Should have 3 keys for single repo (name, pseudoURL, path)")
	})
}
