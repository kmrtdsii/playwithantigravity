package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("config", func() git.Command { return &ConfigCommand{} })
}

type ConfigCommand struct{}

var _ git.Command = (*ConfigCommand)(nil)

func (c *ConfigCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// args[0] is "config"
	if len(args) < 3 {
		return "", fmt.Errorf("usage: git config <key> <value>")
	}

	key := args[1]
	value := strings.Join(args[2:], " ")

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// For now, we just support setting user.name and user.email in the local config
	cfg, err := repo.Config()
	if err != nil {
		return "", err
	}

	switch key {
	case "user.name":
		cfg.User.Name = strings.Trim(value, "'\"")
	case "user.email":
		cfg.User.Email = strings.Trim(value, "'\"")
	default:
		// Ignore other configs or store in raw config?
		// go-git Config struct has specific fields.
		// For generic sections/subsections, it's more complex.
		// For Mission setup, we only care about identity.
	}

	if err := repo.Storer.SetConfig(cfg); err != nil {
		return "", err
	}

	return "", nil
}

func (c *ConfigCommand) Help() string {
	return "usage: git config <key> <value>"
}
