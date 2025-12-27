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

func (c *SwitchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}
	w, _ := repo.Worktree()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git switch [-c] <branch>")
	}

	// Naive parsing for switch for now
	var createBranch string
	target := ""

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-c", "--create":
			if i+1 < len(args) {
				createBranch = args[i+1]
				i++
			}
		case "-h":
			return c.Help(), nil
		default:
			target = arg
		}
	}

	if createBranch != "" {
		// logic for create
		opts := &gogit.CheckoutOptions{
			Create: true,
			Branch: plumbing.ReferenceName("refs/heads/" + createBranch),
		}
		if err := w.Checkout(opts); err != nil {
			return "", err
		}
		s.RecordReflog(fmt.Sprintf("switch: moving to %s", createBranch))
		return fmt.Sprintf("Switched to a new branch '%s'", createBranch), nil
	}

	if target == "" {
		return "", fmt.Errorf("missing branch name")
	}

	err := w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + target),
	})
	if err != nil {
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("switch: moving to %s", target))
	return fmt.Sprintf("Switched to branch '%s'", target), nil
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
