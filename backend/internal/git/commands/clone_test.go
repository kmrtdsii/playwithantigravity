package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestCloneCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-clone")
	cmd := &CloneCommand{}

	t.Run("Clone basic", func(t *testing.T) {
		// Mock URL
		url := "https://github.com/example/repo.git"

		res, err := cmd.Execute(context.Background(), s, []string{"clone", url})
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		if !strings.Contains(res, "Cloned into 'repo'") {
			t.Errorf("Unexpected output: %s", res)
		}

		// Check that repo was created in session
		// clone creates "repo" directory/repo key
		if _, ok := s.Repos["repo"]; !ok {
			t.Error("Repo 'repo' not found in session")
		}

		// Check current dir
		if s.CurrentDir != "/repo" {
			t.Errorf("Expected current dir '/repo', got '%s'", s.CurrentDir)
		}
	})

	t.Run("Clone exists", func(t *testing.T) {
		// Already cloned above
		url := "https://github.com/example/repo.git"
		_, err := cmd.Execute(context.Background(), s, []string{"clone", url})
		if err == nil {
			t.Error("Expected error for existing repo")
		}
	})
}
