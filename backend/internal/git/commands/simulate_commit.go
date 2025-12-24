package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("simulate-commit", func() git.Command { return &SimulateCommitCommand{} })
}

type SimulateCommitCommand struct{}

func (c *SimulateCommitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// Usage: simulate-commit <remote-name> <message> [<author-name> <author-email>]
	if len(args) < 3 {
		return "", fmt.Errorf("usage: simulate-commit <remote-name> <message> [<author-name> <author-email>]")
	}

	remoteName := args[1]
	message := args[2]
	authorName := "Simulated User"
	authorEmail := "simulated@example.com"

	if len(args) >= 5 {
		authorName = args[3]
		authorEmail = args[4]
	}

	sm := s.Manager
	sm.Lock()
	defer sm.Unlock()

	// repo var not needed if using temp clone
	_, ok := sm.SharedRemotes[remoteName]
	if !ok {
		return "", fmt.Errorf("remote %s not found", remoteName)
	}

	// Handle Bare Repos by cloning to temp, committing, and pushing back
	// This is heavier but robust for simulation on bare shared repos.

	// 1. Create Temp Dir
	tempDir, err := os.MkdirTemp("", "gitgym-sim-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// 2. Clone (Local)
	// We clone from the SharedRemote path
	remotePath, ok := sm.SharedRemotePaths[remoteName]
	if !ok {
		// Fallback to iterating if key mismatch, but should match
		return "", fmt.Errorf("remote path for %s not found", remoteName)
	}

	tempRepo, err := gogit.PlainClone(tempDir, false, &gogit.CloneOptions{
		URL: remotePath,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone for simulation: %w", err)
	}

	w, err := tempRepo.Worktree()
	if err != nil {
		return "", fmt.Errorf("temp worktree error: %v", err)
	}

	// 3. Make Change
	filename := fmt.Sprintf("simulated_%d.txt", time.Now().Unix())
	file, err := w.Filesystem.Create(filename)
	if err != nil {
		return "", err
	}
	file.Write([]byte("Simulated content"))
	file.Close()

	if _, err := w.Add(filename); err != nil {
		return "", err
	}

	if authorName == "" {
		authorName = "Simulated User"
	}
	if authorEmail == "" {
		authorEmail = "simulated@example.com"
	}

	hash, err := w.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", err
	}

	// 4. Push back to Shared Remote
	err = tempRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
	})
	if err != nil {
		return "", fmt.Errorf("failed to push simulation: %w", err)
	}

	return fmt.Sprintf("Simulated commit created: %s", hash.String()), nil
}

func (c *SimulateCommitCommand) Help() string {
	return "usage: simulate-commit <remote-name> <message> [<author-name> <author-email>]"
}
