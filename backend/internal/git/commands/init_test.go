package commands

import (
	"context"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestInitCommand_Execute(t *testing.T) {
	// Setup
	fs := memfs.New()
	session := &git.Session{
		Filesystem: fs,
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
	}
	cmd := &InitCommand{}
	ctx := context.Background()

	// Test 1: Init in root should fail
	_, err := cmd.Execute(ctx, session, []string{"init"})
	assert.Error(t, err)

	// Test 2: Init basic
	_, err = cmd.Execute(ctx, session, []string{"init", "repo1"})
	assert.NoError(t, err)
	// Check if repo1 exists in root
	_, exists := session.Repos["repo1"]
	assert.True(t, exists, "repo1 should exist")

	// Test 3: Change dir and init relative
	session.CurrentDir = "/repo1"
	// Create subdir manually first as mkdir would do
	fs.MkdirAll("sub", 0755)

	// Try to init nested repo (should fail due to nested check)
	_, err = cmd.Execute(ctx, session, []string{"init", "sub"})
	assert.Error(t, err, "Should fail verifying nested repo")
	assert.Contains(t, err.Error(), "nested repo exists")

	// Test 4: Init non-nested relative path
	// Go back to root
	session.CurrentDir = "/"
	fs.MkdirAll("folder", 0755)
	session.CurrentDir = "/folder"

	_, err = cmd.Execute(ctx, session, []string{"init", "repo2"})
	assert.NoError(t, err)
	// Check if repo2 exists at folder/repo2
	_, exists = session.Repos["folder/repo2"]
	assert.True(t, exists, "folder/repo2 should exist")
}
