package commands

import (
	"context"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("pwd", func() git.Command { return &PwdCommand{} })
}

type PwdCommand struct{}

// Ensure PwdCommand implements git.Command
var _ git.Command = (*PwdCommand)(nil)

func (c *PwdCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.RLock()
	defer s.RUnlock()

	dir := s.CurrentDir
	if dir == "" {
		return "/", nil
	}
	return dir, nil
}

func (c *PwdCommand) Help() string {
	return `📘 PWD (1)                                              Shell Manual

 💡 DESCRIPTION
    ・「今どこにいるか」（現在のフォルダのパス）を表示する

 📋 SYNOPSIS
    pwd

 🛠  EXAMPLES
    $ pwd
    /gitgym/repo
`
}
