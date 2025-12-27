package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("log", func() git.Command { return &LogCommand{} })
}

type LogCommand struct{}

func (c *LogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	oneline := false

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--oneline":
			oneline = true
		case "-h", "--help":
			return c.Help(), nil
		default:
			// log supports <revision range>, <path>...
			// ignore for now or error?
			// simulated log is simple HEAD traversal
		}
	}

	cIter, err := repo.Log(&gogit.LogOptions{All: false}) // HEAD only usually
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = cIter.ForEach(func(c *object.Commit) error {
		if oneline {
			// 7-char hash + message
			sb.WriteString(fmt.Sprintf("%s %s\n", c.Hash.String()[:7], strings.Split(c.Message, "\n")[0]))
		} else {
			sb.WriteString(fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n\n",
				c.Hash.String(),
				c.Author.Name,
				c.Author.Email,
				c.Author.When.Format(time.RFC3339),
				strings.TrimSpace(c.Message),
			))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (c *LogCommand) Help() string {
	return `📘 GIT-LOG (1)                                          Git Manual

 💡 DESCRIPTION
    ・これまでのコミット履歴（いつ、誰が、何をしたか）を表示する
    ・プロジェクトの歴史を遡って確認する

 📋 SYNOPSIS
    git log [--oneline]

 ⚙️  COMMON OPTIONS
    --oneline
        各コミットを1行（ハッシュの一部とメッセージのみ）で表示します。
        履歴の概観をつかむのに便利です。

 🛠  EXAMPLES
    1. 詳細なログを表示
       $ git log

    2. 簡潔なログを表示
       $ git log --oneline

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-log
`
}
