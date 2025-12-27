package commands

// add.go - Simulated Git Add Command
//
// Stages file contents to the index for the next commit.
// This operates on the in-memory worktree and index.

import (
	"context"
	"fmt"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("add", func() git.Command { return &AddCommand{} })
}

type AddCommand struct{}

func (c *AddCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Flags
	var (
		all bool
	)
	var pathspecs []string

	// Parse flags
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-h", "--help":
			return c.Help(), nil
		case "-A", "--all":
			all = true
		case "--":
			// Remainder are pathspecs
			if i+1 < len(cmdArgs) {
				pathspecs = append(pathspecs, cmdArgs[i+1:]...)
			}
			i = len(cmdArgs) // Break loop
		default:
			if arg == "." {
				all = true // git add . is effectively all in current dir
			}
			pathspecs = append(pathspecs, arg)
		}
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := repo.Worktree()

	if len(pathspecs) == 0 && !all {
		return "", fmt.Errorf("Nothing specified, nothing added.\nMaybe you wanted to say 'git add .'?")
	}

	// Logic
	var err error
	if all {
		// "git add ." or "git add -A"
		// go-git w.Add(".") adds all changes in worktree
		_, err = w.Add(".")
	} else {
		for _, file := range pathspecs {
			_, e := w.Add(file)
			if e != nil {
				return "", e // Error out on first fail? Standard git warns but continues?
				// go-git Add returns err.
			}
		}
	}

	if err != nil {
		return "", err
	}

	if all {
		return "Added changes", nil
	}
	return "Added " + fmt.Sprintf("%v", pathspecs), nil
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
