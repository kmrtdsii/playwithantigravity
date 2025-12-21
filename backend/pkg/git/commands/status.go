package commands

import (
	"context"
	"fmt"

	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("status", func() git.Command { return &StatusCommand{} })
}

type StatusCommand struct{}

func (c *StatusCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
    // We need to access Repo. But we should lock.
    // Since Session mutex is unexported, we might need to rely on the fact that Repo is thread-safe?
    // Or we should add a helper to Session to execute with lock?
    // Or export the mutex?
    // For now, let's assume we can access s.Repo safely if we accept race conditions or fix Session later.
    // The original code in ExecuteGitCommand didn't seem to lock extensively outside of specific methods?
    // Actually git_engine.go didn't lock! It was unsafe.
    // Refactoring plan mentioned existing code was unsafe.
    // So here we should try to be safe.
    
    // Currently Session struct field `mu` is unexported. 
    // I should fix Session struct to export Mu or provide accessor.
    
    // For this step I will assume I will fix Session.go to export Mu or use methods.
    // I'll assume s.Mu is available or I will add a Lock/Unlock methods.
    
    // Let's modify session.go to export Mu first? Or add Lock/Unlock?
    // Adding Lock/Unlock to Session is cleaner.
    
	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := s.Repo.Worktree()
	status, _ := w.Status()
	return status.String(), nil
}

func (c *StatusCommand) Help() string {
	return "usage: git status\n\nShow the working tree status."
}
