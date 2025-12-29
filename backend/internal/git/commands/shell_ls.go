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

// Ensure LsCommand implements git.Command
var _ git.Command = (*LsCommand)(nil)

type LsOptions struct {
	Path     string
	ShowAll  bool // -a flag: show hidden files
	LongList bool // -l flag: long listing (not fully implemented)
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

	// Parse flags and find path argument
	var path string
	for _, arg := range cmdArgs {
		if strings.HasPrefix(arg, "-") {
			// Parse flags
			for _, ch := range arg[1:] {
				switch ch {
				case 'a':
					opts.ShowAll = true
				case 'l':
					opts.LongList = true
				}
			}
		} else {
			path = arg
		}
	}

	if path != "" {
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
	if len(opts.Path) > 1 && strings.HasSuffix(opts.Path, "/") {
		opts.Path = strings.TrimSuffix(opts.Path, "/")
	}

	return opts, nil
}

func (c *LsCommand) executeLs(s *git.Session, opts *LsOptions) (string, error) {
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

		// Skip hidden files unless -a flag is set
		if !opts.ShowAll && strings.HasPrefix(name, ".") {
			continue
		}

		if opts.LongList {
			// -l format: Mode Size ModTime Name
			mode := info.Mode()
			size := info.Size()
			modTime := info.ModTime().Format("Jan 02 15:04")

			// Adjust name for directory
			displayName := name
			if info.IsDir() {
				displayName = name + "/"
			}

			line := fmt.Sprintf("%s %6d %s %s", mode, size, modTime, displayName)
			output = append(output, line)
		} else {
			if info.IsDir() {
				name = name + "/"
			}
			output = append(output, name)
		}
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
    ls [-la] [<path>]

 🛠  OPTIONS
    -a    隠しファイル（.で始まるファイル）も表示
    -l    詳細表示（未実装）

 🛠  EXAMPLES
    1. 現在の場所を表示
       $ ls

    2. 隠しファイルも表示
       $ ls -a
`
}
