package commands

// add.go - Simulated Git Add Command
//
// Stages file contents to the index for the next commit.
// This operates on the in-memory worktree and index.

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("add", func() git.Command { return &AddCommand{} })
}

type AddCommand struct{}

type AddOptions struct {
	All       bool
	Pathspecs []string
}

func (c *AddCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Parse Args
	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	// 2. Resolve Context (Worktree)
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// 3. Execution
	return c.executeAdd(w, opts)
}

func (c *AddCommand) parseArgs(args []string) (*AddOptions, error) {
	opts := &AddOptions{}
	cmdArgs := args[1:]

	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		case "-A", "--all":
			opts.All = true
		case "--":
			// Remainder are pathspecs
			if i+1 < len(cmdArgs) {
				opts.Pathspecs = append(opts.Pathspecs, cmdArgs[i+1:]...)
			}
			return opts, nil // Break entirely as rest are paths
		default:
			if arg == "." {
				opts.All = true
			}
			opts.Pathspecs = append(opts.Pathspecs, arg)
		}
	}
	return opts, nil
}

func (c *AddCommand) executeAdd(w *gogit.Worktree, opts *AddOptions) (string, error) {
	if len(opts.Pathspecs) == 0 && !opts.All {
		return "", fmt.Errorf("Nothing specified, nothing added.\nMaybe you wanted to say 'git add .'?")
	}

	var err error
	if opts.All {
		// "git add ." or "git add -A"
		_, err = w.Add(".")
	} else {
		for _, file := range opts.Pathspecs {
			_, e := w.Add(file)
			if e != nil {
				return "", e
			}
		}
	}

	if err != nil {
		return "", err
	}

	if opts.All {
		return "Added changes", nil
	}
	return "Added " + fmt.Sprintf("%v", opts.Pathspecs), nil
}

func (c *AddCommand) Help() string {
	return `📘 GIT-ADD (1)                                          Git Manual

 💡 DESCRIPTION
    ・変更したファイルをステージングエリア（コミットする準備場所）に追加する
    ・新規作成したファイルをGitの管理対象にする

 📋 SYNOPSIS
    git add [<options>] [--] <pathspec>...

 ⚙️  COMMON OPTIONS
    .
        カレントディレクトリ配下のすべての変更（新規・変更・削除）を追加します。

    -A, --all
        ワークツリー全体のすべての変更を追加します。

 🛠  EXAMPLES
    1. カレントディレクトリのすべての変更をステージング
       $ git add .

    2. 特定のファイルのみをステージング
       $ git add README.md

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-add
`
}
