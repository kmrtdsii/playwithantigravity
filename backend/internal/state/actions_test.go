package state

import (
	"os"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Helper to create a session manager with a temp dir
func setupTestManager(t *testing.T) *SessionManager {
	tmpDir, err := os.MkdirTemp("", "gitgym-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	sm := NewSessionManager()
	sm.DataDir = tmpDir
	return sm
}

func TestIngestRemote_AutoPrune(t *testing.T) {
	// 1. Setup Session Manager & Session
	sm := setupTestManager(t)
	sid := "test-session"
	s, err := sm.CreateSession(sid)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 2. Ingest Remote A
	remoteNameA := "origin"
	// Create a dummy bare repo on disk to act as remote
	// We need a commit for clone to work? gogit Clone might fail on empty repo if Mirror is true?
	// Actually typical git clone warns but works. gogit might be stricter.
	// Let's create a non-bare repo, commit, then use it.
	remoteDiskA, err := os.MkdirTemp("", "remote-a-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(remoteDiskA)

	// Create non-bare to make commit easy
	rInit, err := gogit.PlainInit(remoteDiskA, false)
	if err != nil {
		t.Fatalf("Failed to init remote A: %v", err)
	}
	w, _ := rInit.Worktree()

	// Commit
	msg := "Initial commit"
	_, err = w.Commit(msg, &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Test", Email: "test@example.com"},
		AllowEmptyCommits: true,
	})
	if err != nil {
		t.Fatalf("Failed to commit to remote A: %v", err)
	}

	// Ingest Remote A
	err = sm.IngestRemote(remoteNameA, remoteDiskA)
	if err != nil {
		t.Fatalf("IngestRemote A failed: %v", err)
	}

	// 3. Simulate "Clone" of Remote A
	repoNameA := "repo-a"
	s.Lock()
	// Create dir
	s.Filesystem.MkdirAll(repoNameA, 0755)

	// Determine the stored path for this remote
	// IngestRemote stores it under the URL (which is remoteDiskA here)
	repoPathA, ok := sm.SharedRemotePaths[remoteDiskA]
	if !ok {
		// Fallback checking, though IngestRemote guarantees this if called with URL
		t.Fatalf("Shared remote path not found for %s", remoteDiskA)
	}

	// Create a mock local repo and set its remote to point to repoPathA
	fsA, _ := s.Filesystem.Chroot(repoNameA)
	rA, _ := gogit.Init(memory.NewStorage(), fsA)

	_, _ = rA.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoPathA},
	})
	s.Repos[repoNameA] = rA
	s.CurrentDir = "/" + repoNameA
	s.Unlock()

	// 4. Ingest Remote B
	remoteDiskB, _ := os.MkdirTemp("", "remote-b-*")
	defer os.RemoveAll(remoteDiskB)

	rInitB, err := gogit.PlainInit(remoteDiskB, false)
	if err != nil {
		t.Fatalf("Failed to init remote B: %v", err)
	}
	wB, _ := rInitB.Worktree()
	_, err = wB.Commit("Initial commit B", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Test", Email: "test@example.com"},
		AllowEmptyCommits: true,
	})
	if err != nil {
		t.Fatalf("Failed to commit to remote B: %v", err)
	}

	err = sm.IngestRemote("origin", remoteDiskB)
	if err != nil {
		t.Fatalf("IngestRemote B failed: %v", err)
	}

	// 5. Verify Pruning
	// Pruning is async, so wait a bit
	time.Sleep(100 * time.Millisecond)

	s.RLock()
	if _, exists := s.Repos[repoNameA]; exists {
		t.Errorf("Repo '%s' should have been pruned but still exists", repoNameA)
	}

	// CurrentDir should be /
	if s.CurrentDir != "/" {
		t.Errorf("CurrentDir should be reset to '/', got '%s'", s.CurrentDir)
	}

	// Filesystem check
	if _, err := s.Filesystem.Stat(repoNameA); err == nil {
		t.Errorf("Directory '%s' should have been deleted", repoNameA)
	}
	s.RUnlock()
}
