package git

import (
	"context"
	"fmt"
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
	// Simple aliasing/handling could go here
	factory, ok := registry[cmdName]
	if !ok {
		return "", fmt.Errorf("git: '%s' is not a git command. See 'git --help'.", cmdName)
	}
	
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

// Helper to parse args somewhat consistently if needed
func ParseArgs(args []string) []string {
    // In a real app, use pflag or similar. 
    // For now we just pass raw args to the specific command.
    return args
}
