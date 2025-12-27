package commands

// shell_ls.go - Shell Command: List Directory
//
// This is a SHELL COMMAND (not a git command).
// Lists the contents of the current or specified directory.

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("ls", func() git.Command { return &LsCommand{} })
}

type LsCommand struct{}

type LsOptions struct {
	Path string
}

func (c *LsCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args, s.CurrentDir)
	if err != nil {
		return "", err
	}

	return c.executeLs(s, opts)
}

func (c *LsCommand) parseArgs(args []string, currentDir string) (*LsOptions, error) {
	opts := &LsOptions{}
	cmdArgs := args[1:]

	if len(cmdArgs) > 0 {
		// Only supporting first arg as path for now
		path := cmdArgs[0]
		// Normalize
		if !strings.HasPrefix(path, "/") {
			if currentDir == "/" {
				opts.Path = "/" + path
			} else {
				opts.Path = currentDir + "/" + path
			}
		} else {
			opts.Path = path
		}
	} else {
		opts.Path = currentDir
		if opts.Path == "" {
			opts.Path = "." // fallback
		}
	}
	// Removing trailing slash if present (except proper root)?
	// s.Filesystem usage usually tolerant but clean path is better.
	if len(opts.Path) > 1 && strings.HasSuffix(opts.Path, "/") {
		opts.Path = strings.TrimSuffix(opts.Path, "/")
	}

	return opts, nil
}

func (c *LsCommand) executeLs(s *git.Session, opts *LsOptions) (string, error) {
	// Handle root specially if needed, but assuming path is valid for ReadDir
	// Note: s.Filesystem.ReadDir on "repo" vs "/repo" works differently in memfs?
	// Usually absolute path logic handled by Billy or user.
	// Our session manager "CurrentDir" uses absolute paths like "/repo".

	// Strip leading slash for billy if needed?
	// memfs usually handles root paths or relative?
	// Previous code:
	// path = s.CurrentDir; if path[0] == '/' { path = path[1:] }
	// So it strips leading slash.

	readPath := opts.Path
	if len(readPath) > 0 && readPath[0] == '/' {
		readPath = readPath[1:]
	}
	if readPath == "" {
		readPath = "."
	}

	infos, err := s.Filesystem.ReadDir(readPath)
	if err != nil {
		return "", fmt.Errorf("ls failed: %w", err)
	}

	var output []string
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() {
			name = name + "/"
		}
		output = append(output, name)
	}

	if len(output) == 0 {
		return "", nil
	}
	return strings.Join(output, "\n"), nil
}

func (c *LsCommand) Help() string {
	return `📘 LS (1)                                               Shell Manual

 💡 DESCRIPTION
    ・現在のフォルダにあるファイルやフォルダの一覧を表示する

 📋 SYNOPSIS
    ls [<path>]

 🛠  EXAMPLES
    1. 現在の場所を表示
       $ ls
`
}
