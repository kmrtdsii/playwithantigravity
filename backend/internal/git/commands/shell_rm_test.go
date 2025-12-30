package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestRmCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-rm")
	s.InitRepo("repo")
	s.CurrentDir = "/repo"

	fs := s.Filesystem
	cmd := &RmCommand{}

	// Setup files
	fs.MkdirAll("repo", 0755)
	f, _ := fs.Create("repo/file.txt")
	f.Close()
	fs.MkdirAll("repo/dir1", 0755)
	f, _ = fs.Create("repo/dir1/nested.txt")
	f.Close()

	t.Run("Remove File", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"rm", "file.txt"})
		if err != nil {
			t.Fatalf("rm failed: %v", err)
		}
		if !strings.Contains(res, "Removed file.txt") {
			t.Errorf("Unexpected output: %s", res)
		}
		_, err = fs.Stat("file.txt")
		if err == nil {
			t.Error("file.txt should be gone")
		}
	})

	t.Run("Remove Directory", func(t *testing.T) {
		// rm dir1 usually requires -r if it was strict rm, but current impl implies -rf often?
		// Current Help says: "(implied) -rf".
		res, err := cmd.Execute(context.Background(), s, []string{"rm", "dir1"})
		if err != nil {
			t.Fatalf("rm dir failed: %v", err)
		}
		if !strings.Contains(res, "Removed dir1") {
			t.Errorf("Unexpected output: %s", res)
		}
		_, err = fs.Stat("dir1")
		if err == nil {
			t.Error("dir1 should be gone")
		}
	})

	t.Run("Remove NonExistent", func(t *testing.T) {
		// Implied -rf means no error on missing file
		_, err := cmd.Execute(context.Background(), s, []string{"rm", "nada"})
		if err != nil {
			t.Errorf("Expected nil error for non-existent file path with implied -f, got: %v", err)
		}
	})
}
