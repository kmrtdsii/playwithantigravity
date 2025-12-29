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
		// The path is hashed, so we verify via SharedRemotePaths
		repoPath := sm.SharedRemotePaths[repoName]
		assert.DirExists(t, repoPath)
		assert.True(t, filepath.Base(filepath.Dir(repoPath)) == "remotes")

		// 3. Check session context switch
		// Session needs to be locked/unlocked in real usage, but here we access directly
		assert.Equal(t, "/"+repoName, session.CurrentDir)

		// 4. Check if session filesystem has the directory
		_, err = session.Filesystem.Stat(repoName)
		assert.NoError(t, err, "Directory should be created in session filesystem")

		// 5. Check if repo is initialized in session and has remote
		sessionRepo, ok := session.Repos[repoName]
		assert.True(t, ok, "Session repo should be cached")
		remotes, err := sessionRepo.Remotes()
		assert.NoError(t, err)
		assert.Len(t, remotes, 1)
		assert.Equal(t, "origin", remotes[0].Config().Name)
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

		// Verify CurrentDir updated
		assert.Equal(t, "/"+repoName2, session.CurrentDir)
	})
}
