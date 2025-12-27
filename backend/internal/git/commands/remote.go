package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("remote", func() git.Command { return &RemoteCommand{} })
}

type RemoteCommand struct{}

type RemoteOptions struct {
	SubCmd  string
	Name    string
	URL     string
	Verbose bool
}

func (c *RemoteCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	return c.executeRemote(s, repo, opts)
}

func (c *RemoteCommand) parseArgs(args []string) (*RemoteOptions, error) {
	opts := &RemoteOptions{}
	cmdArgs := args[1:]

	// Pre-scan structure: git remote [-v] [subcmd [args]]
	var positional []string
	for _, arg := range cmdArgs {
		if arg == "-v" || arg == "--verbose" {
			opts.Verbose = true
		} else if arg == "-h" || arg == "--help" {
			return nil, fmt.Errorf("help requested")
		} else if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		}
	}

	if len(positional) > 0 {
		opts.SubCmd = positional[0]
	}
	if len(positional) > 1 {
		opts.Name = positional[1]
	}
	if len(positional) > 2 {
		opts.URL = positional[2]
	}

	return opts, nil
}

func (c *RemoteCommand) executeRemote(_ *git.Session, repo *gogit.Repository, opts *RemoteOptions) (string, error) {
	if opts.SubCmd == "" {
		return listRemotes(repo, opts.Verbose)
	}

	if opts.SubCmd == "add" {
		if opts.Name == "" || opts.URL == "" {
			return "", fmt.Errorf("usage: git remote add <name> <url>")
		}
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: opts.Name,
			URLs: []string{opts.URL},
		})
		if err != nil {
			return "", err
		}
		return "", nil
	}

	if opts.SubCmd == "remove" || opts.SubCmd == "rm" {
		if opts.Name == "" {
			return "", fmt.Errorf("usage: git remote remove <name>")
		}
		err := repo.DeleteRemote(opts.Name)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	return "", fmt.Errorf("unknown subcommand: %s", opts.SubCmd)
}

func listRemotes(repo *gogit.Repository, verbose bool) (string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, r := range remotes {
		cfg := r.Config()
		if verbose {
			for _, url := range cfg.URLs {
				sb.WriteString(fmt.Sprintf("%s\t%s (fetch)\n", cfg.Name, url))
				sb.WriteString(fmt.Sprintf("%s\t%s (push)\n", cfg.Name, url))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", cfg.Name))
		}
	}
	return sb.String(), nil
}

func (c *RemoteCommand) Help() string {
	return `📘 GIT-REMOTE (1)                                       Git Manual

 💡 DESCRIPTION
    リモートリポジトリ（外部の接続先）に関する以下の操作を行います：
    ・登録されている接続先の一覧を表示する（引数なし）
    ・新しい接続先を追加する（add）
    ・不要な接続先を削除する（remove）

 📋 SYNOPSIS
    git remote [-v]
    git remote add <name> <url>
    git remote remove <name>

 ⚙️  COMMON OPTIONS
    -v, --verbose
        URLも含めて詳細に表示します。
`
}
