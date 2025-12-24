package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestSimulateCommitCommand(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// Ingest a remote (init one first)
	remotePath := filepath.Join(tempDir, "remote")
	r, _ := gogit.PlainInit(remotePath, false)
	w, _ := r.Worktree()
	w.Filesystem.Create("readme.txt")
	w.Add("readme.txt")
	w.Commit("Init", &gogit.CommitOptions{Author: &object.Signature{Name: "Me", Email: "me@me.com", When: time.Now()}})

	err := sm.IngestRemote(context.Background(), "origin", remotePath)
	if err != nil {
		t.Fatal(err)
	}

	session, _ := sm.CreateSession("test-session")
	cmd := &SimulateCommitCommand{}

	// Test Execute
	ctx := context.Background()
	output, err := cmd.Execute(ctx, session, []string{"simulate-commit", "origin", "Test Commit"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	t.Log(output)

	// Verify commit exists in shared remote
	sm.RLock()
	repo := sm.SharedRemotes["origin"]
	sm.RUnlock()

	ref, err := repo.Head()
	if err != nil {
		t.Fatal("Remote HEAD not found")
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if commit.Message != "Test Commit" {
		t.Errorf("Expected message 'Test Commit', got '%s'", commit.Message)
	}
}
