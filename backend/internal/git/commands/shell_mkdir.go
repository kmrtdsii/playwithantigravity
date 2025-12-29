package commands

// shell_mkdir.go - Shell Command: Make Directory
//
// This is a SHELL COMMAND (not a git command).
// Creates a new directory in the session filesystem.

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("mkdir", func() git.Command { return &MkdirCommand{} })
}

type MkdirCommand struct{}

// Ensure MkdirCommand implements git.Command
var _ git.Command = (*MkdirCommand)(nil)

func (c *MkdirCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return "", fmt.Errorf("mkdir: missing operand")
	}

	dirName := args[1]

	// Skip flags like -p
	if strings.HasPrefix(dirName, "-") {
		if len(args) < 3 {
			return "", fmt.Errorf("mkdir: missing operand")
		}
		dirName = args[2]
	}

	// Construct full path
	var fullPath string
	if strings.HasPrefix(dirName, "/") {
		// Absolute path
		fullPath = dirName
	} else {
		// Relative path
		if s.CurrentDir == "/" {
			fullPath = "/" + dirName
		} else {
			fullPath = s.CurrentDir + "/" + dirName
		}
	}

	// Strip leading slash for billy filesystem
	fsPath := strings.TrimPrefix(fullPath, "/")
	if fsPath == "" {
		return "", fmt.Errorf("mkdir: cannot create root directory")
	}

	// Check if directory already exists
	if _, err := s.Filesystem.Stat(fsPath); err == nil {
		return "", fmt.Errorf("mkdir: cannot create directory '%s': File exists", dirName)
	}

	// Create the directory
	if err := s.Filesystem.MkdirAll(fsPath, 0755); err != nil {
		return "", fmt.Errorf("mkdir: cannot create directory '%s': %w", dirName, err)
	}

	return "", nil // Success, no output
}

func (c *MkdirCommand) Help() string {
	return `📘 MKDIR (1)                                             Shell Manual

 💡 DESCRIPTION
    ・新しいディレクトリ（フォルダ）を作成する

 📋 SYNOPSIS
    mkdir <directory>

 🛠  EXAMPLES
    1. 新しいリポジトリ用のディレクトリを作成
       $ mkdir my-project
       $ cd my-project
       $ git init
`
}
