package commands

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestRestoreCommand(t *testing.T) {
	// 1. Setup
	sm := git.NewSessionManager()
	// sm.DataDir = t.TempDir() // t.TempDir fails in this env
	tmpDir, err := os.MkdirTemp("/tmp", "gitgym-restore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	sm.DataDir = tmpDir

	session, err := sm.CreateSession("test-restore")
	if err != nil {
		t.Fatal(err)
	}

	// Init Repo
	var repo *gogit.Repository
	repo, err = session.InitRepo("my-repo")
	if err != nil {
		t.Fatal(err)
	}
	session.CurrentDir = "/my-repo" // Important: RestoreCommand uses session.GetRepo() which relies on CurrentDir

	w, _ := repo.Worktree()

	// 2. Prepare files (v1 committed)
	// a.txt
	f, _ := w.Filesystem.Create("a.txt")
	f.Write([]byte("v1"))
	f.Close()
	w.Add("a.txt")

	// b.txt (in subdir)
	w.Filesystem.MkdirAll("subdir", 0755)
	f, _ = w.Filesystem.Create("subdir/b.txt")
	f.Write([]byte("v1-sub"))
	f.Close()
	w.Add("subdir/b.txt")

	w.Commit("Init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// 3. Modify files (v2 - unstaged)
	f, _ = w.Filesystem.OpenFile("a.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	f.Write([]byte("v2-modified"))
	f.Close()

	f, _ = w.Filesystem.OpenFile("subdir/b.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	f.Write([]byte("v2-sub-modified"))
	f.Close()

	// 4. Test: Restore specific file
	cmd := &RestoreCommand{}
	_, err = cmd.Execute(context.Background(), session, []string{"restore", "a.txt"})
	if err != nil {
		t.Errorf("restore a.txt failed: %v", err)
	}

	// Verify a.txt is back to v1
	fRead, _ := w.Filesystem.Open("a.txt")
	content, _ := io.ReadAll(fRead)
	fRead.Close()
	if string(content) != "v1" {
		t.Errorf("Expected 'v1', got '%s'", string(content))
	}

	// Verify b.txt is still modified
	fRead, _ = w.Filesystem.Open("subdir/b.txt")
	content, _ = io.ReadAll(fRead)
	fRead.Close()
	if string(content) != "v2-sub-modified" {
		t.Errorf("Expected b.txt to remain modified, got '%s'", string(content))
	}

	// 5. Test: Restore . (current dir recursive)
	// Modify a.txt again to check if . catches it too
	f, _ = w.Filesystem.OpenFile("a.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	f.Write([]byte("v3-modified"))
	f.Close()

	// Run restore .
	_, err = cmd.Execute(context.Background(), session, []string{"restore", "."})
	if err != nil {
		t.Errorf("restore . failed: %v", err)
	}

	// Verify a.txt restored
	fRead, _ = w.Filesystem.Open("a.txt")
	content, _ = io.ReadAll(fRead)
	fRead.Close()
	if string(content) != "v1" {
		t.Errorf("Expected a.txt 'v1', got '%s'", string(content))
	}

	// Verify b.txt restored (recursive)
	fRead, _ = w.Filesystem.Open("subdir/b.txt")
	content, _ = io.ReadAll(fRead)
	fRead.Close()
	if string(content) != "v1-sub" {
		t.Errorf("Expected subdir/b.txt 'v1-sub', got '%s'", string(content))
	}

	// 6. Test: Restore --staged .
	// Stage changes first
	f, _ = w.Filesystem.OpenFile("a.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	f.Write([]byte("v4-staged"))
	f.Close()
	w.Add("a.txt")

	// Verify staged
	status, _ := w.Status()
	sStat := status.File("a.txt")
	// Note: go-git status codes: ' ' (Unmodified), 'M' (Modified), 'A' (Added), etc.
	// If staged, Staging should be 'M' (if modified from HEAD) or 'A' (if new)
	// Here a.txt existed in HEAD(v1), was reset to v1, then modified to v4 and added.
	// So Staging should be 'M'.
	if sStat.Staging != gogit.Modified {
		t.Errorf("Expected a.txt staging to be Modified, got %c", sStat.Staging)
	}

	// Restore --staged .
	_, err = cmd.Execute(context.Background(), session, []string{"restore", "--staged", "."})
	if err != nil {
		t.Errorf("restore --staged . failed: %v", err)
	}

	status, _ = w.Status()
	// After restore --staged, index should match HEAD (v1).
	// Worktree has "v4-staged" content.
	// So Staging should be ' ' (Unmodified vs HEAD).
	// Worktree should be 'M' (Modified vs Index).

	sStat = status.File("a.txt")
	if sStat.Staging != gogit.Unmodified {
		t.Errorf("Expected a.txt staging to be Unmodified (Unstaged), got %c", sStat.Staging)
	}
	if sStat.Worktree != gogit.Modified {
		t.Errorf("Expected a.txt worktree to be Modified, got %c", sStat.Worktree)
	}
}
