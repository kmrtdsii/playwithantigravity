package git

import (
	"context"
	"fmt"
	"strings"
)

// Command defines the interface for all git commands
type Command interface {
	Execute(ctx context.Context, session *Session, args []string) (string, error)
	Help() string
}

// CommandFactory allows creating new instances of commands
type CommandFactory func() Command

var registry = make(map[string]CommandFactory)

// RegisterCommand registers a command factory
func RegisterCommand(name string, factory CommandFactory) {
	registry[name] = factory
}

// Global dispatcher
func Dispatch(ctx context.Context, session *Session, cmdName string, args []string) (string, error) {
	// All commands (git and shell) are registered in the same registry
	factory, ok := registry[cmdName]
	if !ok {
		return "", fmt.Errorf("'%s' is not a recognized command. See 'help'", cmdName)
	}

	// Clear any simulation/potential commits from previous dry-runs
	session.Lock()
	session.PotentialCommits = nil
	session.Unlock()

	cmd := factory()
	return cmd.Execute(ctx, session, args)
}

// GetSupportedCommands returns all registered commands
func GetSupportedCommands() []string {
	cmds := make([]string, 0, len(registry))
	for k := range registry {
		cmds = append(cmds, k)
	}
	return cmds
}

// GetCommandHelp returns the help string for a command
func GetCommandHelp(name string) (string, error) {
	factory, ok := registry[name]
	if !ok {
		return "", fmt.Errorf("command not found")
	}
	cmd := factory()
	return cmd.Help(), nil
}

// ParseCommand parses the raw input string and returns the resolved command name and arguments.
// It handles aliases like 'add' -> 'git add', 'commit' -> 'git commit', etc.
// The returned args slice always starts with the resolved command name (args[0] == cmdName).
func ParseCommand(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}

	first := parts[0]

	// 1. Check if first word is "git"
	if first == "git" {
		if len(parts) == 1 {
			return "help", []string{"help"}
		}
		
		sub := parts[1]
		
		// Handle global flags as commands or aliases
		switch sub {
		case "-v", "--version":
			return "version", []string{"version"}
		case "-h", "--help":
			return "help", []string{"help"}
		}

		// Block stupid things like "git ls" if "ls" is a shell command valid on its own but not as git subcommand
		// Unless "ls" IS a registered git subcommand (it's not, "ls-files" is).
		// Our registry has "ls" for shell ls.
		// If user types "git ls", they usually mean "git ls-files", but if they mean shell ls, that's invalid syntax.
		// The original code explicitly mapped `ls`, `cd` etc to `git-ls` to fail.
		// We can keep that safeguard or just let it fall through.
		// If we return "ls", dispatch will find "ls" (shell command) and run it.
		// That implies "git ls" runs shell "ls". That is confusing.
		// So we should ONLY return "sub" if "sub" is a GIT command, not a SHELL command?
		// But our registry mixes them.
		
		// Let's preserve the explicit block for now to be safe, or just rely on a naming convention?
		// Actually, simpler: just return the part. Dispatch will run it.
		// If I type "git ls", and "ls" is registered (shell ls), it runs shell ls.
		// Is that bad? "git ls" -> lists files. A bit weird but acceptable for a gym.
		// But existing code wanted to prevent it.
		// Let's stick to the rigid parsing for "git" prefix.
		
		return sub, parts[1:]
	}

	// 2. Check if first word is a command in Registry directly (Aliases like 'commit', 'ls')
	if _, ok := registry[first]; ok {
		// It is a valid command (git or shell)
		// e.g. "commit" -> found in registry -> return "commit"
		// e.g. "ls" -> found -> return "ls"
		
		// One exception: "commit" with args needs to be standard
		if first == "commit" {
			// ensure args[0] is "commit"
			return "commit", append([]string{"commit"}, parts[1:]...)
		}
		
		return first, parts
	}

	// 3. Special cases mapping
	if first == "--version" {
		return "version", []string{"version"}
	}

	// Default fallthrough
	return first, parts
}

// Helper to parse args somewhat consistently if needed
func ParseArgs(args []string) []string {
	// In a real app, use pflag or similar.
	// For now we just pass raw args to the specific command.
	return args
}
