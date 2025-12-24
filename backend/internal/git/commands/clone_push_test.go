package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestPersistentRemoteCycle(t *testing.T) {
	// 1. Setup Directories
	tempDir := t.TempDir()
	sourceRepoPath := filepath.Join(tempDir, "real-github-repo")
	dataDir := filepath.Join(tempDir, "data") // Where "pseudo-remotes" live

	// 2. Create a "Real" Source Repo (simulating GitHub)
	// We init it and add a commit so it's cloneable
	err := os.MkdirAll(sourceRepoPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	realRepo, err := gogit.PlainInit(sourceRepoPath, false)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := realRepo.Worktree()
	readmePath := filepath.Join(sourceRepoPath, "README.md")
	os.WriteFile(readmePath, []byte("# Hello GitHub"), 0644)
	w.Add("README.md")
	firstCommitHash, err := w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. Setup Session Manager
	sm := git.NewSessionManager()
	sm.DataDir = dataDir
	s, _ := sm.CreateSession("user-session")

	// 4. IngestRemote (Simulate Frontend "Update Remote URL")
	// The App downloads "real-github-repo" into "data/origin" (bare)
	err = sm.IngestRemote(context.Background(), "origin", sourceRepoPath)
	if err != nil {
		t.Fatalf("IngestRemote failed: %v", err)
	}

	// Verify Bare Repo Exists
	sm.RLock()
	bareRepoPath, ok := sm.SharedRemotePaths["origin"]
	sm.RUnlock()
	if !ok {
		t.Fatal("SharedRemotePaths['origin'] not set")
	}
	if _, err := os.Stat(bareRepoPath); os.IsNotExist(err) {
		t.Fatalf("Pseudo-remote not created at %s", bareRepoPath)
	}

	// 5. User Clones "origin" (simulated via CloneCommand)
	// The user gives the *real URL* (sourceRepoPath), but the system should map it to *bareRepoPath*
	cloneCmd := &CloneCommand{}
	output, err := cloneCmd.Execute(context.Background(), s, []string{"clone", sourceRepoPath})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}
	t.Log(output)

	// Verify Local Repo in Session
	// Check origin URL
	localRepo := s.Repos["real-github-repo"]
	if localRepo == nil {
		t.Fatal("Local repo not found in session")
	}
	remote, err := localRepo.Remote("origin")
	if err != nil {
		t.Fatal(err)
	}
	originURL := remote.Config().URLs[0]
	// It should point to bareRepoPath
	// We check if it contains "data/origin"
	// Note: exact string match might fail due to absolute path differences on some OS, but expected behavior is absolute path to dataDir/origin
	// it should point to sourceRepoPath (friendly URL) now, masked from internal path
	if originURL != sourceRepoPath {
		t.Errorf("FAIL: Origin points to '%s', expected friendly URL '%s'", originURL, sourceRepoPath)
	}
	t.Logf("Cloned from: %s", originURL)

	// 6. User Pushes a New Commit
	// Create local commit
	localW, _ := localRepo.Worktree()
	// Since we are using memfs for session, we access it via s.Filesystem or the worktree abstraction
	// The CloneCommand sets up s.Filesystem chrooted.
	// We need to write a file in the session filesystem.
	// Wait, s.Repos uses memfs.

	newFile, _ := localW.Filesystem.Create("new-feature.txt")
	newFile.Write([]byte("Amazing feature"))
	newFile.Close()
	localW.Add("new-feature.txt")
	_, err = localW.Commit("Add feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "user@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Push Command (We need to instantiate PushCommand or call remote.Push directly?)
	// Let's use the actual PushCommand if available to test full flow
	// If not easy, we manually push on repo.
	// Checking available commands... PushCommand exists.

	// We need to register it or instantiate it.
	// git.RegisterCommand("push", ...)
	// Let's manually instantiate for test
	pushCmd := &PushCommand{}
	// PushCommand usually expects "push [remote] [branch]"
	// Defaults to "origin" "current-branch"
	// We are on main/master.

	pushOutput, err := pushCmd.Execute(context.Background(), s, []string{"push"})
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	t.Log(pushOutput)

	// 7. Verify the Push reached the Pseudo-Remote (Bare Repo)
	// Open the bare repo
	bareRepo, _ := gogit.PlainOpen(bareRepoPath)
	// Check HEAD
	ref, err := bareRepo.Head()
	if err != nil {
		t.Fatal(err)
	}
	if ref.Hash() == firstCommitHash {
		t.Fatal("Pseudo-remote HEAD did not move. Push failed to update remote.")
	}

	// 8. Verify the Push did NOT reach the Real GitHub Repo
	// The real repo should still be at firstCommitHash
	realHead, _ := realRepo.Head()
	if realHead.Hash() != firstCommitHash {
		t.Fatal("Real repo was updated! Implementation leaked push to upstream.")
	}

	t.Log("Success: Push updated pseudo-remote but not real upstream.")
}
