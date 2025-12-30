package commands

import (
	"context"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestAddCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-add")
	cmd := &AddCommand{}

	// Init manually since "git init" command is disabled
	_, _ = s.InitRepo("repo")
	s.CurrentDir = "/repo"

	repo := s.GetRepo()
	w, _ := repo.Worktree()

	t.Run("Add File", func(t *testing.T) {
		f, _ := w.Filesystem.Create("test.txt")
		_, _ = f.Write([]byte("content"))
		_ = f.Close()

		res, err := cmd.Execute(context.Background(), s, []string{"add", "test.txt"})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if !strings.Contains(res, "Added") {
			t.Errorf("Unexpected output: %s", res)
		}

		status, _ := w.Status()
		if status.IsClean() {
			t.Error("Status should not be clean after add")
		}
		if status.File("test.txt").Staging != gogit.Added {
			t.Errorf("File not staged as Added")
		}
	})

	t.Run("Add All", func(t *testing.T) {
		f, _ := w.Filesystem.Create("test2.txt")
		_, _ = f.Write([]byte("content2"))
		_ = f.Close()

		res, err := cmd.Execute(context.Background(), s, []string{"add", "."})
		if err != nil {
			t.Fatalf("Add all failed: %v", err)
		}
		if !strings.Contains(res, "Added") {
			t.Errorf("Unexpected output: %s", res)
		}

		status, _ := w.Status()
		if status.File("test2.txt").Staging != gogit.Added {
			t.Errorf("File test2.txt not staged")
		}
	})
}
