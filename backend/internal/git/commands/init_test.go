package commands

import (
	"context"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSession creates a fresh session for testing
func newTestSession() *git.Session {
	return &git.Session{
		Filesystem: memfs.New(),
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
	}
}

func TestInitCommand_Execute(t *testing.T) {
	cmd := &InitCommand{}
	ctx := context.Background()

	t.Run("InitInRootShouldFail", func(t *testing.T) {
		session := newTestSession()
		_, err := cmd.Execute(ctx, session, []string{"init"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot init repository at root")
	})

	t.Run("InitWithNameFromRoot", func(t *testing.T) {
		session := newTestSession()
		result, err := cmd.Execute(ctx, session, []string{"init", "myrepo"})
		require.NoError(t, err)
		assert.Contains(t, result, "Initialized empty Git repository")

		_, exists := session.Repos["myrepo"]
		assert.True(t, exists, "myrepo should exist in session.Repos")
	})

	t.Run("InitRelativePathFromSubdirectory", func(t *testing.T) {
		session := newTestSession()
		// First create a folder structure
		require.NoError(t, session.Filesystem.MkdirAll("projects", 0755))
		session.CurrentDir = "/projects"

		result, err := cmd.Execute(ctx, session, []string{"init", "webapp"})
		require.NoError(t, err)
		assert.Contains(t, result, "/projects/webapp/.git/")

		_, exists := session.Repos["projects/webapp"]
		assert.True(t, exists, "projects/webapp should exist")
	})

	t.Run("InitAbsolutePath", func(t *testing.T) {
		session := newTestSession()
		session.CurrentDir = "/somedir"

		result, err := cmd.Execute(ctx, session, []string{"init", "/absoluterepo"})
		require.NoError(t, err)
		assert.Contains(t, result, "/absoluterepo/.git/")

		_, exists := session.Repos["absoluterepo"]
		assert.True(t, exists, "absoluterepo should exist")
	})

	t.Run("NestedRepoInsideExistingShouldFail", func(t *testing.T) {
		session := newTestSession()
		// Create parent repo first
		_, err := cmd.Execute(ctx, session, []string{"init", "parent"})
		require.NoError(t, err)

		// Try to create nested repo
		session.CurrentDir = "/parent"
		_, err = cmd.Execute(ctx, session, []string{"init", "child"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot init repository inside existing repo")
	})

	t.Run("ParentOfExistingRepoShouldFail", func(t *testing.T) {
		session := newTestSession()
		// Create child repo first
		require.NoError(t, session.Filesystem.MkdirAll("outer/inner", 0755))
		session.CurrentDir = "/outer/inner"
		_, err := cmd.Execute(ctx, session, []string{"init"})
		require.NoError(t, err)

		// Try to create parent repo that would contain child
		session.CurrentDir = "/"
		_, err = cmd.Execute(ctx, session, []string{"init", "outer"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nested repo exists")
	})

	t.Run("ReinitExistingRepoShouldFail", func(t *testing.T) {
		session := newTestSession()
		_, err := cmd.Execute(ctx, session, []string{"init", "samerepo"})
		require.NoError(t, err)

		// Try to reinitialize
		_, err = cmd.Execute(ctx, session, []string{"init", "samerepo"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository already exists")
	})

	t.Run("PathWithDotDot", func(t *testing.T) {
		session := newTestSession()
		require.NoError(t, session.Filesystem.MkdirAll("a/b", 0755))
		session.CurrentDir = "/a/b"

		// ../c should resolve to /a/c
		result, err := cmd.Execute(ctx, session, []string{"init", "../c"})
		require.NoError(t, err)
		assert.Contains(t, result, "/a/c/.git/")

		_, exists := session.Repos["a/c"]
		assert.True(t, exists, "a/c should exist")
	})
}

func TestInitCommand_Help(t *testing.T) {
	cmd := &InitCommand{}
	help := cmd.Help()
	assert.Contains(t, help, "git init")
	assert.Contains(t, help, "directory")
}
