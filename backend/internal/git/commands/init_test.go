package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestInitCommand_Execute(t *testing.T) {
	sm := git.NewSessionManager()
	sessionID := "test-init"
	s, _ := sm.CreateSession(sessionID)

	cmd := &InitCommand{}

	// 1. Test init at root (default behavior)
	s.CurrentDir = "/"
	res, err := cmd.Execute(context.Background(), s, []string{"init"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if res == "" {
		t.Error("Expected non-empty response")
	}

	// 2. Test init with argument
	s2, _ := sm.CreateSession("test-init-arg")
	res, err = cmd.Execute(context.Background(), s2, []string{"init", "myrepo"})
	if err != nil {
		t.Fatalf("Execute with arg failed: %v", err)
	}
	if res == "" {
		t.Error("Expected non-empty response for init with arg")
	}
	if _, ok := s2.Repos["myrepo"]; !ok {
		t.Error("Repo 'myrepo' not found in session")
	}

	// 3. Test double init
	res, err = cmd.Execute(context.Background(), s2, []string{"init", "myrepo"})
	if err != nil {
		t.Fatalf("Double init failed: %v", err)
	}
	if !strings.Contains(res, "Git repository already initialized") {
		t.Errorf("Expected already initialized message, got: %s", res)
	}
}
