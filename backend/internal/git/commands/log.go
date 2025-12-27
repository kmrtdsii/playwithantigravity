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

// Ensure LogCommand implements git.Command
var _ git.Command = (*LogCommand)(nil)

type LogOptions struct {
	Oneline bool
	Args    []string // Revisions or paths
}

func (c *LogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
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

	return c.executeLog(s, repo, opts)
}

func (c *LogCommand) parseArgs(args []string) (*LogOptions, error) {
	opts := &LogOptions{}
	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		switch arg {
		case "--oneline":
			opts.Oneline = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			opts.Args = append(opts.Args, arg)
		}
	}
	return opts, nil
}

func (c *LogCommand) executeLog(_ *git.Session, repo *gogit.Repository, opts *LogOptions) (string, error) {
	// TODO: support revision range in opts.Args if needed.
	// Current simulation uses default HEAD traversal.

	cIter, err := repo.Log(&gogit.LogOptions{All: false}) // HEAD only usually
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = cIter.ForEach(func(c *object.Commit) error {
		if opts.Oneline {
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
