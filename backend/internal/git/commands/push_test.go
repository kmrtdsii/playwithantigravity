package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

// setupPushTestSession creates a session with:
// - A local repo "localrepo" with one commit
// - A simulated remote "remoterepo" (bare) registered as shared remote
// - The local repo configured with remote "origin" pointing to "/remoterepo"
func setupPushTestSession(t *testing.T, sm *git.SessionManager, id string) *git.Session {
	s, _ := sm.CreateSession(id)
	ctx := context.Background()

	// 1. Create local repo
	initCmd := &InitCommand{}
	_, err := initCmd.Execute(ctx, s, []string{"init", "localrepo"})
	if err != nil {
		t.Fatalf("setup: init localrepo failed: %v", err)
	}
	s.CurrentDir = "/localrepo"

	// Create a file and commit
	touchCmd := &TouchCommand{}
	_, _ = touchCmd.Execute(ctx, s, []string{"touch", "file.txt"})
	addCmd := &AddCommand{}
	_, _ = addCmd.Execute(ctx, s, []string{"add", "."})
	commitCmd := &CommitCommand{}
	_, err = commitCmd.Execute(ctx, s, []string{"commit", "-m", "Initial commit"})
	if err != nil {
		t.Fatalf("setup: commit failed: %v", err)
	}

	// 2. Create remote repo (simulated)
	s.CurrentDir = "/"
	_, err = initCmd.Execute(ctx, s, []string{"init", "remoterepo"})
	if err != nil {
		t.Fatalf("setup: init remoterepo failed: %v", err)
	}

	// Register remoterepo as shared remote for lookup by push command
	sm.SharedRemotes["remoterepo"] = s.Repos["remoterepo"]

	// 3. Add remote to local repo
	s.CurrentDir = "/localrepo"
	remoteCmd := &RemoteCommand{}
	_, err = remoteCmd.Execute(ctx, s, []string{"remote", "add", "origin", "/remoterepo"})
	if err != nil {
		t.Fatalf("setup: remote add failed: %v", err)
	}

	return s
}

func TestPushCommand_Help(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupPushTestSession(t, sm, "test-push-help")
	ctx := context.Background()

	cmd := &PushCommand{}
	res, err := cmd.Execute(ctx, s, []string{"push", "--help"})
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(res, "usage:") {
		t.Errorf("Expected help text, got: %s", res)
	}
	if !strings.Contains(res, "--force") {
		t.Errorf("Expected help to include --force option, got: %s", res)
	}
}

func TestPushCommand_DryRun(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupPushTestSession(t, sm, "test-push-dryrun")
	ctx := context.Background()

	cmd := &PushCommand{}
	// Push without specifying branch (uses HEAD)
	res, err := cmd.Execute(ctx, s, []string{"push", "--dry-run", "origin"})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !strings.Contains(res, "[dry-run]") {
		t.Errorf("Expected dry-run message, got: %s", res)
	}
}

func TestPushCommand_BasicPush(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupPushTestSession(t, sm, "test-push-basic")
	ctx := context.Background()

	cmd := &PushCommand{}
	// Push default branch (HEAD)
	res, err := cmd.Execute(ctx, s, []string{"push", "origin"})
	if err != nil {
		t.Fatalf("basic push failed: %v", err)
	}
	if res == "" {
		t.Error("Expected non-empty output")
	}

	// Verify remote now has the branch (go-git defaults to 'main')
	remoteRepo := sm.SharedRemotes["remoterepo"]
	if remoteRepo == nil {
		t.Fatal("remoterepo not found")
	}
	// Try main first, fallback to master
	_, err = remoteRepo.Reference("refs/heads/main", true)
	if err != nil {
		_, err = remoteRepo.Reference("refs/heads/master", true)
		if err != nil {
			t.Errorf("remote should have the pushed branch after push: %v", err)
		}
	}
}

func TestPushCommand_NoRemote(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-push-noremote")
	ctx := context.Background()

	// Create a repo without remote
	initCmd := &InitCommand{}
	_, _ = initCmd.Execute(ctx, s, []string{"init", "norepo"})
	s.CurrentDir = "/norepo"

	touchCmd := &TouchCommand{}
	_, _ = touchCmd.Execute(ctx, s, []string{"touch", "file.txt"})
	addCmd := &AddCommand{}
	_, _ = addCmd.Execute(ctx, s, []string{"add", "."})
	commitCmd := &CommitCommand{}
	_, _ = commitCmd.Execute(ctx, s, []string{"commit", "-m", "Initial"})

	cmd := &PushCommand{}
	_, err := cmd.Execute(ctx, s, []string{"push", "origin"})
	if err == nil {
		t.Error("Expected error for missing remote")
	}
}
