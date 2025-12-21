package commands

import (
	"context"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("init", func() git.Command { return &InitCommand{} })
}

type InitCommand struct{}

func (c *InitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()
	// Note: s.Mu is not exported in current session.go, I need to check capitalization. 
    // It was `mu sync.RWMutex` (unexported).
    // I need to properly lock via methods or export the mutex?
    // Session operations usually lock themselves?
    // InitSession logic in git_engine.go just set s.Repo.
    
    // Check if repo exists
	if s.Repo != nil {
		return "Git repository already initialized", nil
	}

	st := memory.NewStorage()
	repo, err := gogit.Init(st, s.Filesystem)
	if err != nil {
		return "", err
	}
	s.Repo = repo

	// Set default branch to main
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
	err = s.Repo.Storer.SetReference(headRef)
    if err != nil {
        return "", err
    }

	return "Initialized empty Git repository in /", nil
}

func (c *InitCommand) Help() string {
	return "usage: git init\n\nInitialize a new git repository."
}
