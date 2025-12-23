package commands

// help.go - Git Help Command
//
// Displays help information for available commands.
// Usage: git help [<command>]

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("help", func() git.Command { return &HelpCommand{} })
}

type HelpCommand struct{}

func (c *HelpCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	// help doesn't need to lock session usually, unless it accesses something.
	// It relies on registry.

	if len(args) > 1 {
		subcmd := args[1]
		// Create a temporary instance to call Help()
		// We can't access registry directly if it's private in pkg/git.
		// We need git.GetCommand(name)? Or git.Dispatch just executes.
		// We added git.GetSupportedCommands in engine.go.
		// We probably need a way to get the Help string for a command.
		// Let's modify engine.go to expose GetHelp(cmdName).

		// For now, I can't easily access the Help() method of other commands without instantiating them via the registry which is private.
		// I should add git.GetCommandHelp(name) to engine.go.

		// Assuming I will add git.GetCommandHelp(name)
		helpStr, err := git.GetCommandHelp(subcmd)
		if err != nil {
			return fmt.Sprintf("git help: unknown command '%s'", subcmd), nil
		}
		return helpStr, nil
	}

	cmds := git.GetSupportedCommands()
	sort.Strings(cmds)

	var sb strings.Builder
	sb.WriteString("Supported commands:\n")
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("   %s\n", cmd))
	}
	sb.WriteString("\nType 'git help <command>' for more information about a specific command.")
	return sb.String(), nil
}

func (c *HelpCommand) Help() string {
	return "usage: git help [<command>]\n\nDisplay help information."
}
