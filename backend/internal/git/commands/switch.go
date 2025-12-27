package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("switch", func() git.Command { return &SwitchCommand{} })
}

// SwitchCommand is similar but strictly for branches
type SwitchCommand struct{}

type SwitchOptions struct {
	CreateBranch string
	TargetBranch string
}

func (c *SwitchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	return c.executeSwitch(s, w, opts)
}

func (c *SwitchCommand) parseArgs(args []string) (*SwitchOptions, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: git switch [-c] <branch>")
	}
	opts := &SwitchOptions{}
	cmdArgs := args[1:]

	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-c", "--create":
			if i+1 < len(cmdArgs) {
				opts.CreateBranch = cmdArgs[i+1]
				i++
			}
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			opts.TargetBranch = arg
		}
	}
	return opts, nil
}

func (c *SwitchCommand) executeSwitch(s *git.Session, w *gogit.Worktree, opts *SwitchOptions) (string, error) {
	if opts.CreateBranch != "" {
		// logic for create
		checkoutOpts := &gogit.CheckoutOptions{
			Create: true,
			Branch: plumbing.ReferenceName("refs/heads/" + opts.CreateBranch),
		}
		if err := w.Checkout(checkoutOpts); err != nil {
			return "", err
		}
		s.RecordReflog(fmt.Sprintf("switch: moving to %s", opts.CreateBranch))
		return fmt.Sprintf("Switched to a new branch '%s'", opts.CreateBranch), nil
	}

	if opts.TargetBranch == "" {
		return "", fmt.Errorf("missing branch name")
	}

	err := w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + opts.TargetBranch),
	})
	if err != nil {
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("switch: moving to %s", opts.TargetBranch))
	return fmt.Sprintf("Switched to branch '%s'", opts.TargetBranch), nil
}

func (c *SwitchCommand) Help() string {
	return `📘 GIT-SWITCH (1)                                       Git Manual

 💡 DESCRIPTION
    ・作業するブランチを切り替える
    ・新しいブランチを作成して、そのまま切り替える（-c）

 📋 SYNOPSIS
    git switch <branch>
    git switch -c <new-branch>

 ⚙️  COMMON OPTIONS
    -c, --create <new-branch>
        新しいブランチを作成して切り替えます（` + "`" + `git checkout -b` + "`" + ` 相当）。

 🛠  EXAMPLES
    1. ブランチを切り替え
       $ git switch main

    2. 作成して切り替え
       $ git switch -c new-feature
`
}
