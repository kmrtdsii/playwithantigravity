package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("init", func() git.Command { return &InitCommand{} })
}

type InitCommand struct{}

func (c *InitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	var dir string
	isBare := false

	// Parse args
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--bare" {
			isBare = true
		} else if !strings.HasPrefix(arg, "-") {
			dir = arg
		}
	}

	// Resolve target path
	targetPath := s.CurrentDir
	if dir != "" {
		// handle relative/absolute logic simplified
		if strings.HasPrefix(dir, "/") {
			targetPath = dir
		} else {
			if s.CurrentDir == "/" || s.CurrentDir == "" {
				targetPath = dir
			} else {
				targetPath = s.CurrentDir + "/" + dir
			}
		}
	}

	// Normalize (remove leading slash for map key if needed, or consistent usage)
	// Our session keys usually don't have leading slash if relative to root of MemFS?
	// s.CurrentDir normally has leading slash for display, but Filesystem root is "".
	// Let's ensure proper pathing for chroot.
	cleanPath := targetPath
	if len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}

	if cleanPath != "" {
		if err := s.Filesystem.MkdirAll(cleanPath, 0755); err != nil {
			return "", err
		}
	}

	// Check if registered
	if _, ok := s.Repos[cleanPath]; ok {
		return fmt.Sprintf("Git repository already initialized in %s", targetPath), nil
	}

	repoFS, err := s.Filesystem.Chroot(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to chroot: %w", err)
	}

	st := memory.NewStorage()
	var repo *gogit.Repository
	if isBare {
		repo, err = gogit.Init(st, nil) // Bare repo has no worktree (filesystem)
		// But wait, Init(st, nil) might not use `repoFS` for config?
		// Actually, standard Init with nil filesystem implies bare?
		// gogit.Init(st, nil) -> Bare
		// gogit.Init(st, fs) -> Non-Bare
	} else {
		repo, err = gogit.Init(st, repoFS)
	}

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

	typeStr := ""
	if isBare {
		typeStr = "bare "
	}

	return fmt.Sprintf("Initialized empty %sGit repository in /%s", typeStr, cleanPath), nil
}

func (c *InitCommand) Help() string {
	return "usage: git init [--bare] [directory]"
}
