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

	// Handle Aliases
	switch parts[0] {
	case "reset", "add", "merge", "tag", "rebase", "checkout", "branch", "switch":
		// Standard git commands that might be typed without 'git' prefix in some contexts
		// or just passed directly.
		return parts[0], append([]string{parts[0]}, parts[1:]...)
	case "commit":
		// commit -m ... -> git commit -m ...
		// ensure args[0] is "commit"
		return "commit", append([]string{"commit"}, parts[1:]...)
	case "--version":
		return "version", []string{"version"}
	}

	// Default handling
	// If it starts with "git", the registry key is parts[1] (e.g. "git clone" -> "clone")
	if parts[0] == "git" {
		if len(parts) == 1 {
			return "help", []string{"help"}
		}
		if len(parts) > 1 {
			cmd := parts[1]
			switch cmd {
			case "-v", "--version":
				return "version", []string{"version"}
			case "-h", "--help":
				return "help", []string{"help"}
			}

			// Prevent "git <shell_cmd>" from mapping to shell commands
			// The user explicitly stated pwd, cd, ls, touch are not git commands.
			// git rm is a valid git command (though currently simulated by shell rm), so we exclude it from this block.
			switch parts[1] {
			case "ls", "cd", "pwd", "touch":
				// Return a name that won't match, e.g. "git-ls"
				// This causes Dispatch to return "unknown command"
				return "git-" + parts[1], parts
			}

			return parts[1], parts[1:]
		}
	}

	// Otherwise, assume the first word is the command (e.g. "ls", "rm", "cd")
	return parts[0], parts
}

// Helper to parse args somewhat consistently if needed
func ParseArgs(args []string) []string {
	// In a real app, use pflag or similar.
	// For now we just pass raw args to the specific command.
	return args
}
