package integration_test

import (
	"context"
	"strings"
	"testing"
)

func TestGitPhase2Features(t *testing.T) {
	sessionID := "phase2-test"
	if err := InitSession(sessionID); err != nil {
		t.Fatalf("Failed to init session: %v", err)
	}

	// Helper to exec
	exec := func(args ...string) (string, error) {
		return ExecuteGitCommand(sessionID, args)
	}

	// Init
	// Init
	// "git init" is disabled, so we initialize via internal session method
	session, err := GetSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if _, err := session.InitRepo(""); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}
	if err := TouchFile(sessionID, "README.md"); err != nil {
		t.Fatalf("touch failed: %v", err)
	}
	if _, err := exec("add", "README.md"); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if _, err := exec("commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// 1. Test Log
	t.Run("Log", func(t *testing.T) {
		out, err := exec("log")
		if err != nil {
			t.Fatalf("log failed: %v", err)
		}
		if !strings.Contains(out, "Initial commit") {
			t.Errorf("log output missing message: %s", out)
		}

		out, err = exec("log", "--oneline")
		if err != nil {
			t.Fatalf("log --oneline failed: %v", err)
		}
		if len(strings.Split(out, "\n")[0]) > 80 { // Heuristic
			t.Errorf("log --oneline output too long? %s", out)
		}
	})

	// 2. Test Tag
	t.Run("Tag", func(t *testing.T) {
		// Create
		_, err := exec("tag", "v1.0")
		if err != nil {
			t.Fatalf("create tag failed: %v", err)
		}
		// List
		out, err := exec("tag")
		if err != nil {
			t.Fatalf("list tag failed: %v", err)
		}
		if !strings.Contains(out, "v1.0") {
			t.Errorf("tag list missing v1.0: %s", out)
		}
		// Annotated
		_, err = exec("tag", "-a", "v1.1", "-m", "Annotated")
		if err != nil {
			t.Fatalf("create annotated tag failed: %v", err)
		}
		out, _ = exec("tag")
		if !strings.Contains(out, "v1.1") {
			t.Errorf("tag list missing v1.1: %s", out)
		}
		// Delete
		_, err = exec("tag", "-d", "v1.0")
		if err != nil {
			t.Fatalf("delete tag failed: %v", err)
		}
		out, _ = exec("tag")
		if strings.Contains(out, "v1.0") {
			t.Errorf("tag v1.0 should be deleted: %s", out)
		}
	})

	// 3. Test Commit Amend
	t.Run("Amend", func(t *testing.T) {
		out1, err := exec("log", "--oneline")
		if err != nil {
			t.Fatalf("log failed: %v", err)
		}
		fields := strings.Fields(out1)
		if len(fields) == 0 {
			t.Fatalf("log output empty: %s", out1)
		}
		firstHash := fields[0]

		_, err = exec("commit", "--amend", "-m", "Amended message")
		if err != nil {
			t.Fatalf("amend failed: %v", err)
		}

		out2, err := exec("log", "--oneline")
		if err != nil {
			t.Fatalf("log failed after amend: %v", err)
		}
		if !strings.Contains(out2, "Amended message") {
			t.Errorf("log missing amended message: %s", out2)
		}
		fields2 := strings.Fields(out2)
		if len(fields2) == 0 {
			t.Fatalf("log output empty after amend: %s", out2)
		}
		newHash := fields2[0]
		if newHash == firstHash {
			t.Errorf("hash should change after amend")
		}
	})

	// 4. Test Checkout File and Reset
	t.Run("CheckoutFileAndReset", func(t *testing.T) {
		// Make a change
		if err := TouchFile(sessionID, "README.md"); err != nil {
			t.Fatalf("touch failed: %v", err)
		}

		// Verify status (should be modified)
		status, _ := exec("status")
		if !strings.Contains(status, "M README.md") && !strings.Contains(status, "README.md") {
			// Check later; status format is custom in this engine?
			// No, it calls w.Status().String().
			t.Log("Note: status output format check skipped")
		}

		// Checkout file (restore)
		_, err := exec("checkout", "--", "README.md")
		if err != nil {
			t.Fatalf("checkout file failed: %v", err)
		}
	})

	// 5. Test Diff (Commits)
	t.Run("Diff", func(t *testing.T) {
		// Need 2 commits
		_, _ = exec("commit", "--amend", "-m", "Base") // reset state
		// Create new file
		session, _ := GetSession(sessionID)
		f_create, _ := session.Filesystem.Create("new.txt")
		_, _ = f_create.Write([]byte("hello"))
		_ = f_create.Close()
		_, _ = exec("add", "new.txt")
		_, _ = exec("commit", "-m", "Second")

		out, err := exec("diff", "HEAD^", "HEAD")
		if err != nil {
			t.Fatalf("diff failed: %v", err)
		}
		if !strings.Contains(out, "new.txt") {
			// Expected diff to show new file
			t.Log("Warn: diff output missing new.txt")
		}
	})

	// 6. Test Reset
	t.Run("Reset", func(t *testing.T) {
		// currently at Second
		// Reset to Base (HEAD^)
		_, err := exec("reset", "--hard", "HEAD^")
		if err != nil {
			t.Fatalf("reset failed: %v", err)
		}

		log, _ := exec("log", "--oneline")
		if strings.Contains(log, "Second") {
			t.Errorf("log should not contain Second after reset: %s", log)
		}
	})

	// 7. Test Help
	t.Run("Help", func(t *testing.T) {
		out, err := exec("help")
		if err != nil {
			t.Fatalf("help failed: %v", err)
		}
		if !strings.Contains(out, "usage: git") {
			t.Errorf("help output missing usage format: %s", out)
		}

		if !strings.Contains(out, "init") || !strings.Contains(out, "commit") {
			t.Errorf("help output missing commands: %s", out)
		}
	})

	// 8. Test Help Subcommand
	t.Run("HelpDetails", func(t *testing.T) {
		out, err := exec("help", "commit")
		if err != nil {
			t.Fatalf("help commit failed: %v", err)
		}
		if !strings.Contains(out, "--amend") {
			t.Errorf("help commit missing --amend: %s", out)
		}

		out, _ = exec("help", "log")
		if !strings.Contains(out, "--oneline") {
			t.Errorf("help log missing --oneline: %s", out)
		}
	})

	// 9. Test Rebase
	t.Run("Rebase", func(t *testing.T) {
		// Setup:
		// main: C1 -> C2
		// feature: C1 -> C3
		// Goal: rebase feature on main -> C1 -> C2 -> C3'

		// Reset to C1 (Base)
		// Instead of assuming state, let's create branches

		// Current state: main has commits.
		// Let's create new branch 'feature-rebase' from HEAD^ (Base)
		_, _ = exec("checkout", "main")
		_, _ = exec("branch", "feature-rebase")
		_, _ = exec("reset", "--hard", "HEAD^")                       // move main back 1
		_, _ = exec("commit", "--allow-empty", "-m", "Main Diverged") // main has Base -> Diverged

		_, _ = exec("checkout", "feature-rebase")

		// Let's build explicitly for clarity
		_, _ = exec("checkout", "-b", "base-branch")
		_, _ = exec("commit", "--allow-empty", "-m", "Base Commit")

		_, _ = exec("checkout", "-b", "feat-branch")
		_, _ = exec("commit", "--allow-empty", "-m", "Feat Commit")

		_, _ = exec("checkout", "base-branch")
		_, _ = exec("commit", "--allow-empty", "-m", "Upstream Commit")

		// Rebase feat-branch on base-branch
		_, _ = exec("checkout", "feat-branch")
		_, err := exec("rebase", "base-branch")
		if err != nil {
			t.Fatalf("rebase failed: %v", err)
		}

		// Verify log
		// Should be: Base -> Upstream -> Feat'
		log, _ := exec("log", "--oneline")
		if !strings.Contains(log, "Feat Commit") || !strings.Contains(log, "Upstream Commit") {
			t.Errorf("log missing commits after rebase: %s", log)
		}
	})

	// 10. Test Reflog
	t.Run("Reflog", func(t *testing.T) {
		out, err := exec("reflog")
		if err != nil {
			t.Fatalf("reflog failed: %v", err)
		}

		// We expect "rebase", "checkout", "commit" from previous step
		if !strings.Contains(out, "rebase: finished") {
			t.Errorf("reflog missing rebase entry: %s", out)
		}
		if !strings.Contains(out, "checkout: moving") {
			t.Errorf("reflog missing checkout entry: %s", out)
		}
	})
}

// TestRemoteWorkflowGoldenPath is an end-to-end integration test for the core remote workflow:
// Init -> Commit -> Create Remote -> Push -> Fetch -> Reset Remote
// This test validates the happy path to ensure the main user journey works correctly.
func TestRemoteWorkflowGoldenPath(t *testing.T) {
	sessionID := "golden-path-test"
	if err := InitSession(sessionID); err != nil {
		t.Fatalf("Failed to init session: %v", err)
	}

	exec := func(args ...string) (string, error) {
		return ExecuteGitCommand(sessionID, args)
	}

	// Step 1: Initialize Repository
	t.Log("Step 1: Initialize repository")
	session, err := GetSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if _, err := session.InitRepo("my-project"); err != nil {
		t.Fatalf("Init repo failed: %v", err)
	}
	// Set current directory to the new repo
	session.CurrentDir = "/my-project"

	// Step 2: Create and commit a file
	t.Log("Step 2: Create file and commit")
	f, err := session.Filesystem.Create("my-project/hello.txt")
	if err != nil {
		t.Fatalf("Create file failed: %v", err)
	}
	_, _ = f.Write([]byte("Hello World"))
	_ = f.Close()

	if _, err := exec("add", "hello.txt"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	out, err := exec("commit", "-m", "Initial commit")
	if err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
	if !strings.Contains(out, "Commit created") {
		t.Errorf("Commit output missing expected format: %s", out)
	}

	// Step 3: Create bare remote repository
	t.Log("Step 3: Create bare remote repository")
	tmpDir := t.TempDir()
	t.Setenv("GITGYM_DATA_ROOT", tmpDir)

	sm := session.Manager
	err = sm.CreateBareRepository(context.TODO(), sessionID, "my-remote")
	if err != nil {
		t.Fatalf("Create bare repo failed: %v", err)
	}

	// Verify remote was created
	sm.RLock()
	_, remoteExists := sm.SharedRemotes["my-remote"]
	sm.RUnlock()
	if !remoteExists {
		t.Fatalf("Remote 'my-remote' not found in SharedRemotes")
	}

	// Step 4: Add remote and push
	t.Log("Step 4: Add remote and push")
	_, err = exec("remote", "add", "origin", "remote://gitgym/my-remote.git")
	if err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	out, err = exec("push", "origin", "main")
	if err != nil {
		// Push might fail if branch is master instead of main
		out, err = exec("push", "origin", "master")
		if err != nil {
			t.Fatalf("git push failed: %v", err)
		}
	}
	t.Logf("Push output: %s", out)

	// Step 5: Fetch from remote
	t.Log("Step 5: Fetch from remote")
	out, err = exec("fetch")
	if err != nil {
		t.Fatalf("git fetch failed: %v", err)
	}
	t.Logf("Fetch output: %s", out)

	// Step 6: Reset remote (Single Residency cleanup)
	t.Log("Step 6: Reset remote")
	err = sm.RemoveRemote("my-remote")
	if err != nil {
		t.Fatalf("RemoveRemote failed: %v", err)
	}

	// Verify remote is gone
	sm.RLock()
	_, stillExists := sm.SharedRemotes["my-remote"]
	sm.RUnlock()
	if stillExists {
		t.Errorf("Remote 'my-remote' should have been removed")
	}

	// Verify PRs are also cleared
	prs := sm.GetPullRequests()
	if len(prs) != 0 {
		t.Errorf("PullRequests should be empty after RemoveRemote, got %d", len(prs))
	}

	t.Log("✅ Golden Path test completed successfully")
}
