package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// setupPushTestSession creates a session with:
// - A local repo "localrepo" with one commit
// - A simulated remote "remoterepo" (bare) registered as shared remote
// - The local repo configured with remote "origin" pointing to "/remoterepo"
func setupPushTestSession(_ *testing.T, sm *git.SessionManager, id string) *git.Session {
	// Create Session first
	s, _ := sm.CreateSession(id)

	// 1. Create local repo directly
	fs := memfs.New()
	st := memory.NewStorage()
	r, _ := gogit.Init(st, fs)
	s.Repos = map[string]*gogit.Repository{"localrepo": r} // Manually register
	s.CurrentDir = "/localrepo"
	// Also need to set Filesystem in Session if commands use it?
	// commands usually use s.GetRepo() -> returns s.Repos[dir]
	// s.Filesystem is usually the one mounted at s.CurrentDir?
	// Session has a global Filesystem? No, Session has `Filesystem billy.Filesystem`.
	// If we use multiple repos in one session, usually we mount them or use chroot?
	// Simplified: Set s.Filesystem to this repo's fs
	s.Filesystem = fs

	// Create a file and commit
	// Use worktree directly to avoid command overhead/restrictions
	w, _ := r.Worktree()
	f, _ := w.Filesystem.Create("file.txt")
	f.Write([]byte("content"))
	f.Close()
	w.Add("file.txt")
	w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// 2. Create remote repo (simulated)
	// Just create another memory repo and register in SharedRemotes
	remoteSt := memory.NewStorage()
	remoteFs := memfs.New() // Bare usually? Or normal. Push works to bare or non-bare (if configured).
	// gogit supports pushing to non-bare, but might fail to update checked out branch if not configured.
	// Let's make it bare for simplicity or just normal.
	remoteRepo, _ := gogit.Init(remoteSt, remoteFs)

	// Register remoterepo as shared remote
	sm.SharedRemotes["remoterepo"] = remoteRepo

	// s.Repos["remoterepo"] is needed if we use internal path resolution?
	// The original code used s.Repos for "remoterepo" too.
	s.Repos["remoterepo"] = remoteRepo

	// 3. Add remote to local repo
	r.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"/remoterepo"},
	})

	// Ensure HEAD is on a branch (master)
	// gogit.Init creates master by default.

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
	if !strings.Contains(res, "SYNOPSIS") {
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
	_, _ = s.InitRepo("norepo")
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
