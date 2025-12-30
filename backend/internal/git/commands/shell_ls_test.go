package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestLsCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-ls")
	s.InitRepo("repo")
	s.CurrentDir = "/repo"

	fs := s.Filesystem
	cmd := &LsCommand{}

	// Setup files
	fs.MkdirAll("repo/dir1", 0755)
	f, _ := fs.Create("repo/file1.txt")
	f.Close()
	f, _ = fs.Create("repo/file2.txt")
	f.Close()
	f, _ = fs.Create("repo/dir1/nested.txt")
	f.Close()

	t.Run("Ls Current Dir", func(t *testing.T) {
		res, err := cmd.Execute(context.Background(), s, []string{"ls"})
		if err != nil {
			t.Fatalf("ls failed: %v", err)
		}
		if !strings.Contains(res, "file1.txt") || !strings.Contains(res, "dir1/") {
			t.Errorf("Unexpected output: %s", res)
		}
	})

	t.Run("Ls Specific Dir", func(t *testing.T) {
		// Currently broken in impl, expecting fail or fix
		res, err := cmd.Execute(context.Background(), s, []string{"ls", "dir1"})
		if err != nil {
			t.Fatalf("ls dir1 failed: %v", err)
		}
		if !strings.Contains(res, "nested.txt") {
			t.Errorf("Expected nested.txt in dir1 listing, got: %s", res)
		}
	})
}
