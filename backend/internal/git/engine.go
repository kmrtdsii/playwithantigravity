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
	case "reset":
		return "reset", append([]string{"reset"}, parts[1:]...)
	case "add":
		return "add", append([]string{"add"}, parts[1:]...)
	case "commit":
		// commit -m ... -> commit -m ...
		// Original alias was "git commit -m". The command expects "commit" as args[0].
		// However, we need to inject "-m" if it was part of the simplification?
		// simple alias: commit "msg" -> git commit -m "msg"
		// Let's stick to simple normalization: args[0] = "commit".
		// If the alias logic added flags (like -m), we should add them.
		// Previous logic: newParts := []string{"git", "commit", "-m"}
		// So args (for Dispatch) was ["git", "commit", "-m", ...]
		// Removing "git": ["commit", "-m", ...]
		return "commit", append([]string{"commit", "-m"}, parts[1:]...)
	case "merge":
		return "merge", append([]string{"merge"}, parts[1:]...)
	case "tag":
		return "tag", append([]string{"tag"}, parts[1:]...)
	case "rebase":
		return "rebase", append([]string{"rebase"}, parts[1:]...)
	case "checkout":
		return "checkout", append([]string{"checkout"}, parts[1:]...)
	case "branch":
		return "branch", append([]string{"branch"}, parts[1:]...)
	case "switch":
		return "switch", append([]string{"switch"}, parts[1:]...)
		// Note: The original handler mapped these to "git <cmd>".
		// However, the Dispatch function expects "cmdName" to be the key in the registry.
		// Most git commands are registered as their name (e.g. "add", "commit")?
		// Let's check init() in commands.
		// e.g. commands/add.go -> RegisterCommand("add", ...)
		// So "git add" -> args[0]="git", args[1]="add".
		//
		// Wait, the previous handler logic was:
		// case "add": newParts := []string{"git", "add"}
		// then: if parts[0] == "git" { cmdName = parts[1] }
		//
		// So the Dispatcher expects the *subcommand* name if it is a git command.
		// BUT, non-git commands like "ls" are registered as "ls".
		//
		// To unify this, ParseCommand should return the REGISTRY key and the FULL args.
	}

	// Default handling
	// If it starts with "git", the registry key is parts[1] (e.g. "git clone" -> "clone")
	if parts[0] == "git" && len(parts) > 1 {
		// Return parts[1] as name, and parts[1:] as args (so args[0] == name)
		return parts[1], parts[1:]
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
