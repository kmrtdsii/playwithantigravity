package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestCleanCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-clean")

	// Init
	s.InitRepo("repo")
	s.CurrentDir = "/repo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Helper to create untracked file
	createUntracked := func(name string) {
		f, _ := w.Filesystem.Create(name)
		f.Write([]byte("untracked"))
		f.Close()
	}

	// Helper to create untracked dir
	createUntrackedDir := func(name string) {
		w.Filesystem.MkdirAll(name, 0755)
		f, _ := w.Filesystem.Create(name + "/file.txt")
		f.Write([]byte("inside dir"))
		f.Close()
	}

	cmd := &CleanCommand{}

	t.Run("Clean Failure No Force", func(t *testing.T) {
		createUntracked("u1.txt")
		_, err := cmd.Execute(context.Background(), s, []string{"clean"})
		if err == nil {
			t.Fatal("expected failure without force")
		}
		if !strings.Contains(err.Error(), "refusing to clean") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Clean Dry Run", func(t *testing.T) {
		createUntracked("u2.txt")
		res, err := cmd.Execute(context.Background(), s, []string{"clean", "-n"})
		if err != nil {
			t.Fatalf("dry run failed: %v", err)
		}
		if !strings.Contains(res, "Would remove u2.txt") {
			t.Errorf("expected would remove u2.txt, got: %s", res)
		}
		// File should still exist
		_, err = w.Filesystem.Stat("u2.txt")
		if err != nil {
			t.Error("u2.txt should still exist")
		}
	})

	t.Run("Clean Force", func(t *testing.T) {
		// Use u1.txt and u2.txt from previous runs if they still exist (u1 yes, u2 yes)
		res, err := cmd.Execute(context.Background(), s, []string{"clean", "-f"})
		if err != nil {
			t.Fatalf("clean force failed: %v", err)
		}
		if !strings.Contains(res, "Removing u1.txt") {
			t.Errorf("expected removing u1.txt, got: %s", res)
		}

		_, err = w.Filesystem.Stat("u1.txt")
		if err == nil {
			t.Error("u1.txt should be gone")
		}
	})

	t.Run("Clean Directory", func(t *testing.T) {
		createUntrackedDir("dir1")

		// 1. clean -f (should NOT remove dir, but remove file inside)
		res, err := cmd.Execute(context.Background(), s, []string{"clean", "-f"})
		_ = res // use result or ignore explicitly
		if err != nil {
			t.Fatal(err)
		}

		_, err = w.Filesystem.Stat("dir1")
		if err != nil {
			t.Error("dir1 should still exist without -d")
		}

		// 2. clean -fd on NEW uncleaned dir
		createUntrackedDir("dir2")
		res, err = cmd.Execute(context.Background(), s, []string{"clean", "-fd"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(res, "Removing dir2") {
			t.Errorf("expected Removing dir2, got: %s", res)
		}

		_, err = w.Filesystem.Stat("dir2")
		if err == nil {
			t.Error("dir2 should be gone with -fd")
		}
	})
}
