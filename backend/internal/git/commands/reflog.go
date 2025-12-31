package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("reflog", func() git.Command { return &ReflogCommand{} })
}

type ReflogCommand struct{}

// Ensure ReflogCommand implements git.Command
var _ git.Command = (*ReflogCommand)(nil)

func (c *ReflogCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse flags
	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		switch arg {
		case "-h", "--help":
			return c.Help(), nil
		default:
			// reflog usually takes subcommand "show", "expire", "delete", "exists"
			// default is "show"
			// implementing simple loop for help support
		}
	}

	var sb strings.Builder
	// Git reflog shows newest first (HEAD@{0} is current)
	count := len(s.Reflog)
	for i := count - 1; i >= 0; i-- {
		entry := s.Reflog[i]
		// index 0 is oldest, so HEAD@{count-1-i} ?? No.
		// Standard: HEAD@{0} is the most recent (last appended).
		// So i (which is index in slice) corresponds to what?
		// if slice is [A, B, C], C is newest (HEAD@{0}).
		// i=2 (C) -> 0
		// i=1 (B) -> 1
		// i=0 (A) -> 2
		// Formula: count - 1 - i

		refIndex := count - 1 - i

		sb.WriteString(fmt.Sprintf("%s HEAD@{%d}: %s\n", entry.Hash[:7], refIndex, entry.Message))
	}
	return sb.String(), nil
}

func (c *ReflogCommand) Help() string {
	return `📘 GIT-REFLOG (1)                                       Git Manual

 💡 DESCRIPTION
    ・HEAD（現在の場所）の移動履歴を表示する
    ・間違ってリセットしてしまった場合の復元ポイントを探す

 📋 SYNOPSIS
    git reflog

 🛠  EXAMPLES
    1. HEADの履歴を表示
       $ git reflog

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-reflog
`
}
