package commands

import (
	"context"
	"fmt"

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

    path := s.CurrentDir
    cleanPath := path
    if len(path) > 0 && path[0] == '/' {
        cleanPath = path[1:]
    }

    if cleanPath == "" {
         // Allow init with argument? git init myrepo
         if len(args) > 1 {
             newName := args[1]
             cleanPath = newName
             if err := s.Filesystem.MkdirAll(cleanPath, 0755); err != nil {
                 return "", err
             }
         } else {
             return "", fmt.Errorf("cannot initialize repository at workspace root")
         }
    }

    // Check if registered
    if _, ok := s.Repos[cleanPath]; ok {
        return "Git repository already initialized", nil
    }

	// Double check if existing repo logic in go-git needs chroot?
    // We want to init IN that folder.
    repoFS, err := s.Filesystem.Chroot(cleanPath)
    if err != nil {
        return "", fmt.Errorf("failed to chroot: %w", err)
    }

	st := memory.NewStorage()
	repo, err := gogit.Init(st, repoFS)
	if err != nil {
		return "", err
	}
	s.Repos[cleanPath] = repo

	// Set default branch to main
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
	err = repo.Storer.SetReference(headRef)
    if err != nil {
        return "", err
    }

	return fmt.Sprintf("Initialized empty Git repository in /%s", cleanPath), nil
}


func (c *InitCommand) Help() string {
	return "usage: git init\n\nInitialize a new git repository."
}
