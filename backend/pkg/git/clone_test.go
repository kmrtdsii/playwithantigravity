package git_test

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

		// Verify repo state
		session, _ := GetSession(sessionID)
		if session.Repo == nil {
			t.Fatal("session.Repo is nil after clone")
		}

		// Verify HEAD exists
		ref, err := session.Repo.Head()
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

	// 2. Test Push Restriction
	t.Run("PushRestriction", func(t *testing.T) {
		_, err := exec("push", "origin", "main")
		if err == nil {
			t.Error("push command succeeded unexpectedly")
		}
		
		expectedError := "'push' is not a git command"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("unexpected error message: %v (expected to contain %q)", err, expectedError)
		}
	})
}
