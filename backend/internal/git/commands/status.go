package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("status", func() git.Command { return &StatusCommand{} })
}

type StatusCommand struct{}

type StatusOptions struct {
	Short bool
}

func (c *StatusCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	return c.executeStatus(s, repo, opts)
}

func (c *StatusCommand) parseArgs(args []string) (*StatusOptions, error) {
	opts := &StatusOptions{}
	// status command doesn't have many flags in simulation yet, but prepare structure
	for _, arg := range args[1:] {
		switch arg {
		case "-s", "--short":
			opts.Short = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		}
	}
	return opts, nil
}

func (c *StatusCommand) executeStatus(s *git.Session, repo *gogit.Repository, opts *StatusOptions) (string, error) {
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	status, err := w.Status()
	if err != nil {
		return "", err
	}

	// TODO: Implement actual Short status format if opts.Short is true.
	// For now defaulting to standard output string.
	// go-git Status.String() is somewhat verbose/standard.

	return status.String(), nil
}

func (c *StatusCommand) Help() string {
	return `📘 GIT-STATUS (1)                                       Git Manual

 💡 DESCRIPTION
    ・「どのファイルが変更されたか」を確認する
    ・「どのファイルがコミット準備できているか」を確認する
    ・現在のブランチや状況を確認する

 📋 SYNOPSIS
    git status

 🛠  EXAMPLES
    1. 現状を確認する
       $ git status

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-status
`
}
