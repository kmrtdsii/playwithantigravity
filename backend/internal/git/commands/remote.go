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

func (c *RemoteCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Syntax:
	// git remote [-v]
	// git remote add <name> <url>
	// git remote remove <name>

	if len(args) == 1 {
		// List remotes
		// Default: just names
		// git remote -v: names + urls
		return listRemotes(repo, false)
	}

	subCmd := args[1]
	if subCmd == "-v" {
		return listRemotes(repo, true)
	}

	if subCmd == "add" {
		if len(args) < 4 {
			return "", fmt.Errorf("usage: git remote add <name> <url>")
		}
		name := args[2]
		url := args[3]
		_, err := repo.CreateRemote(&config.RemoteConfig{
			Name: name,
			URLs: []string{url},
		})
		if err != nil {
			return "", err
		}
		return "", nil // git remote add is silent on success
	}

	if subCmd == "remove" || subCmd == "rm" {
		if len(args) < 3 {
			return "", fmt.Errorf("usage: git remote remove <name>")
		}
		name := args[2]
		err := repo.DeleteRemote(name)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	return "", fmt.Errorf("unknown subcommand: %s", subCmd)
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
	return `usage: git remote [-v]
       git remote add <name> <url>
       git remote remove <name>

Manage set of tracked repositories.`
}
