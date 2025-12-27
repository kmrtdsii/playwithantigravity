package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("show", func() git.Command { return &ShowCommand{} })
}

type ShowCommand struct{}

type ShowOptions struct {
	NameStatus bool
	CommitID   string
}

func (c *ShowCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
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

	return c.executeShow(s, repo, opts)
}

func (c *ShowCommand) parseArgs(args []string) (*ShowOptions, error) {
	opts := &ShowOptions{
		CommitID: "HEAD", // Default
	}
	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		if arg == "--name-status" {
			opts.NameStatus = true
		} else if strings.HasPrefix(arg, "--format=") {
			// ignore
		} else if arg == "-h" || arg == "--help" {
			return nil, fmt.Errorf("help requested")
		} else if !strings.HasPrefix(arg, "-") {
			opts.CommitID = arg
		}
	}
	return opts, nil
}

func (c *ShowCommand) executeShow(s *git.Session, repo *gogit.Repository, opts *ShowOptions) (string, error) {
	h, err := repo.ResolveRevision(plumbing.Revision(opts.CommitID))
	if err != nil {
		return "", err
	}

	commit, err := repo.CommitObject(*h)
	if err != nil {
		return "", err
	}

	if !opts.NameStatus {
		// Fallback to basic commit info
		return commit.String(), nil
	}

	// Calculate Diff with Parent
	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return "", err
		}
		parentTree, err = parent.Tree()
		if err != nil {
			return "", err
		}
	}

	currentTree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	var changes object.Changes
	if parentTree != nil {
		changes, err = parentTree.Diff(currentTree)
	} else {
		// Root diff
		return listRootChanges(currentTree)
	}

	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue
		}

		var status string
		var path string

		switch action {
		case merkletrie.Insert:
			status = "A"
			path = change.To.Name
		case merkletrie.Delete:
			status = "D"
			path = change.From.Name
		case merkletrie.Modify:
			status = "M"
			path = change.To.Name
		default:
			status = "M"
			path = change.To.Name
		}

		sb.WriteString(fmt.Sprintf("%s\t%s\n", status, path))
	}

	return sb.String(), nil
}

func (c *ShowCommand) Help() string {
	return `📘 GIT-SHOW (1)                                         Git Manual

 💡 DESCRIPTION
    ・特定のコミットの変更内容やメッセージを詳しく表示する
    ・コミットの内容を詳細に確認する

 📋 SYNOPSIS
    git show [<commit>] [--name-status]

 ⚙️  COMMON OPTIONS
    --name-status
        変更内容の差分テキストではなく、変更されたファイル名と状態（A/M/D）のみを表示します。

 🛠  EXAMPLES
    1. 最新のコミットを表示
       $ git show

    2. 特定のコミットの変更ファイル一覧を表示
       $ git show --name-status e5a3b21

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-show
`
}

func listRootChanges(tree *object.Tree) (string, error) {
	var sb strings.Builder
	err := tree.Files().ForEach(func(f *object.File) error {
		sb.WriteString(fmt.Sprintf("A\t%s\n", f.Name))
		return nil
	})
	return sb.String(), err
}
