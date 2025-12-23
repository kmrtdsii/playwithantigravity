package integration_test

import (
	"strings"
	"testing"
	"time"
)

func TestGitCloneAndPushRestriction(t *testing.T) {
	// Create a unique session ID for clone test
	sessionID := "clone-test-" + time.Now().Format("20060102150405")
	if err := InitSession(sessionID); err != nil {
		t.Fatalf("Failed to init session: %v", err)
	}

	exec := func(args ...string) (string, error) {
		return ExecuteGitCommand(sessionID, args)
	}

	// 1. Test Clone
	t.Run("Clone", func(t *testing.T) {
		// Using a small public repo
		repoUrl := "https://github.com/git-fixtures/basic.git"
		out, err := exec("clone", repoUrl)
		if err != nil {
			t.Fatalf("clone failed: %v", err)
		}
		if !strings.Contains(out, "Cloned into") {
			t.Errorf("unexpected output: %s", out)
		}

		// Verify Clone automatically changed directory
		session, _ := GetSession(sessionID)
		expectedDir := "/basic"
		if session.CurrentDir != expectedDir {
			t.Errorf("Expected CurrentDir to be %s, got %s", expectedDir, session.CurrentDir)
		}

		// Verify repo state
		if session.GetRepo() == nil {
			t.Fatal("session.GetRepo() is nil after clone")
		}

		// Verify HEAD exists
		ref, err := session.GetRepo().Head()
		if err != nil {
			t.Fatalf("failed to get HEAD: %v", err)
		}
		if ref == nil {
			t.Fatal("HEAD is nil")
		}

		// Verify file existence (basic.git has specific files)
		// It has a LICENSE file usually, or just check log
		logOut, err := exec("log", "--oneline")
		if err != nil {
			t.Fatalf("log failed after clone: %v", err)
		}
		if len(logOut) == 0 {
			t.Error("log is empty after clone")
		}
	})

	// 2. Test Push Simulation (Should succeed for simulated remotes)
	t.Run("PushSimulation", func(t *testing.T) {
		// Get current branch name
		session, _ := GetSession(sessionID)
		headRef, _ := session.GetRepo().Head()
		branchName := headRef.Name().Short()

		// Create a small change to push
		if _, err := exec("commit", "--allow-empty", "-m", "Simulation push test"); err != nil {
			t.Fatalf("failed to create commit for push: %v", err)
		}

		out, err := exec("push", "origin", branchName)
		if err != nil {
			t.Fatalf("push command failed unexpectedly: %v", err)
		}

		if !strings.Contains(out, "To /remotes/basic.git") {
			t.Errorf("unexpected push output: %s", out)
		}

		// Verify that the remote repo (simulated) now has the commit
		remoteRepo := session.Repos["remotes/basic.git"]
		if remoteRepo == nil {
			t.Fatal("simulated remote repo not found in session")
		}

		// Check if remote branch matches local HEAD
		localRef, _ := session.GetRepo().Head()
		remoteRef, err := remoteRepo.Reference(headRef.Name(), true)
		if err != nil {
			t.Fatalf("failed to find branch %s in remote: %v", branchName, err)
		}

		if remoteRef.Hash() != localRef.Hash() {
			t.Errorf("remote hash %s does not match local hash %s", remoteRef.Hash(), localRef.Hash())
		}
	})
}
