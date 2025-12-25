package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// Helper to create a commit on a specific branch
func commitFile(t *testing.T, r *gogit.Repository, filename, content, msg string) {
	w, _ := r.Worktree()
	f, _ := w.Filesystem.Create(filename)
	f.Write([]byte(content))
	f.Close()
	w.Add(filename)
	_, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
}

func TestPull_Merge(t *testing.T) {
	// 1. Setup Remote
	remoteFs := memfs.New()
	remoteStore := memory.NewStorage()
	remoteRepo, _ := gogit.Init(remoteStore, remoteFs)

	// Base commit (Common Ancestor)
	commitFile(t, remoteRepo, "base.txt", "base content", "Initial commit")

	// 2. Setup Client Session
	sm := git.NewSessionManager()
	sm.DataDir = t.TempDir()

	// Register the remote as "origin" (simulated via SharedRemotes usually, or we can just use file path but memfs is tricky)
	// For simplicity in test, we can use a local path remote or rely on SessionManager's Ingest if we use physical files.
	// Since PullCommand uses `sm.Manager.SharedRemotes` OR `session.Repos`, we can inject into session.Repos.
	// We need a key. "origin" is usually a URL.
	// Let's use a dummy URL "https://example.com/repo.git"

	session, _ := sm.CreateSession("test-pull-merge")
	remoteURL := "https://example.com/repo.git"

	// Inject directly into SharedRemotes (mocking Ingest)
	sm.SharedRemotes[remoteURL] = remoteRepo

	// Clone first involves adding the remote. But `git clone` does that.
	// Let's manually setup the local repo to be in a "cloned" state.
	localFs := memfs.New()
	localStore := memory.NewStorage()
	localRepo, _ := gogit.Clone(localStore, localFs, &gogit.CloneOptions{
		URL: remoteURL, // This won't work with go-git Clone unless we mock it or use file path
	})
	// Wait, standard go-git Clone requires network.
	// Our `CloneCommand` handles the simulation logic!
	// So we should use `CloneCommand`.

	cloneCmd := &CloneCommand{}
	_, err := cloneCmd.Execute(context.Background(), session, []string{"clone", remoteURL})
	if err != nil {
		t.Fatalf("setup: clone failed: %v", err)
	}
	// Note: CloneCommand puts the repo into session.Repos["repo"] and sets session.CurrentDir

	// Get the local repo object from session
	// The repo name is "repo" (basename of URL)
	// session.CurrentDir is "/repo"

	localRepo = session.GetRepo()
	if localRepo == nil {
		t.Fatal("local repo not found in session")
	}

	// 3. Create Divergence

	// A. Update Remote (Theirs)
	commitFile(t, remoteRepo, "remote_file.txt", "remote content", "Remote commit")

	// B. Update Local (Ours) - on master
	commitFile(t, localRepo, "local_file.txt", "local content", "Local commit")

	// 4. Pull
	cmd := &PullCommand{}
	output, err := cmd.Execute(context.Background(), session, []string{"pull", "origin"}) // default merges origin/master
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}

	t.Logf("Pull output: %s", output)

	// 5. Verify
	// Check if both files exist
	w, _ := localRepo.Worktree()
	_, err = w.Filesystem.Stat("remote_file.txt")
	if err != nil {
		t.Errorf("remote_file.txt missing after merge")
	}
	_, err = w.Filesystem.Stat("local_file.txt")
	if err != nil {
		t.Errorf("local_file.txt missing after merge")
	}

	// Verify it's a merge commit (2 parents)
	head, _ := localRepo.Head()
	headCommit, _ := localRepo.CommitObject(head.Hash())
	if len(headCommit.ParentHashes) != 2 {
		t.Errorf("Expected 2 parents for merge commit, got %d", len(headCommit.ParentHashes))
	}
}

func TestPull_Conflict(t *testing.T) {
	// 1. Setup Remote
	remoteFs := memfs.New()
	remoteStore := memory.NewStorage()
	remoteRepo, _ := gogit.Init(remoteStore, remoteFs)

	// Base commit
	commitFile(t, remoteRepo, "file.txt", "base content\n", "Initial commit")

	// 2. Setup Client
	sm := git.NewSessionManager()
	sm.DataDir = t.TempDir()

	remoteURL := "https://example.com/conflict.git"
	sm.SharedRemotes[remoteURL] = remoteRepo

	cloneCmd := &CloneCommand{}
	session, _ := sm.CreateSession("test-pull-conflict")
	_, err := cloneCmd.Execute(context.Background(), session, []string{"clone", remoteURL})
	if err != nil {
		t.Fatalf("setup: clone failed: %v", err)
	}
	localRepo := session.GetRepo()

	// 3. Create Conflict

	// A. Update Remote (Change same line)
	commitFile(t, remoteRepo, "file.txt", "remote changes\n", "Remote update")

	// B. Update Local
	commitFile(t, localRepo, "file.txt", "local changes\n", "Local update")

	// 4. Pull
	cmd := &PullCommand{}
	output, err := cmd.Execute(context.Background(), session, []string{"pull"})
	if err != nil {
		t.Fatalf("pull execution returned error (should handle conflict gracefully?): %v", err)
	}

	t.Logf("Pull output: %s", output)

	if !strings.Contains(output, "CONFLICT") {
		t.Errorf("Expected conflict message, got: %s", output)
	}

	// 5. Verify Conflict Markers
	w, _ := localRepo.Worktree()
	f, _ := w.Filesystem.Open("file.txt")
	content := make([]byte, 100)
	n, _ := f.Read(content)
	fileStr := string(content[:n])

	if !strings.Contains(fileStr, "<<<<<<< HEAD") {
		t.Errorf("Conflict markers missing in file.txt: %s", fileStr)
	}
}
