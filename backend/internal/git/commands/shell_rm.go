package commands

// shell_rm.go - Shell Command: Remove File/Directory
//
// This is a SHELL COMMAND (not a git command).
// Removes files or directories from the simulated filesystem.

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("rm", func() git.Command { return &RmCommand{} })
}

type RmCommand struct{}

// Ensure RmCommand implements git.Command
var _ git.Command = (*RmCommand)(nil)

type RmOptions struct {
	Recursive bool
	Force     bool
	Paths     []string
}

func (c *RmCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	return c.executeRm(s, opts)
}

func (c *RmCommand) parseArgs(args []string) (*RmOptions, error) {
	opts := &RmOptions{
		Recursive: true, // Defaulting to implied -rf as per existing behavior
		Force:     true,
	}
	cmdArgs := args[1:]

	for _, arg := range cmdArgs {
		if strings.HasPrefix(arg, "-") {
			// Parse flags if provided, though defaults are already true.
			// This enables standard usage patterns to be valid.
			if strings.Contains(arg, "r") || strings.Contains(arg, "R") {
				opts.Recursive = true
			}
			if strings.Contains(arg, "f") {
				opts.Force = true
			}
			// Should we support disabling? No standard flag for that.
		} else {
			opts.Paths = append(opts.Paths, arg)
		}
	}

	if len(opts.Paths) == 0 {
		return nil, fmt.Errorf("usage: rm [-rf] <path>")
	}
	return opts, nil
}

func (c *RmCommand) executeRm(s *git.Session, opts *RmOptions) (string, error) {
	var removed []string

	for _, path := range opts.Paths {
		// Safety check: Don't allow deleting root or critical paths if possible
		if path == "/" || path == "." || path == ".." {
			continue
		}

		// Normalize path
		targetPath := path
		if !strings.HasPrefix(targetPath, "/") {
			if s.CurrentDir == "/" {
				targetPath = "/" + targetPath
			} else {
				targetPath = s.CurrentDir + "/" + path
			}
		}

		// Check if it exists
		fi, err := s.Filesystem.Stat(targetPath)
		if err != nil {
			if opts.Force {
				continue // rm -f ignores missing files
			}
			return "", fmt.Errorf("cannot remove '%s': No such file or directory", path)
		}

		// Check if it is a directory representing a repo
		if fi.IsDir() {
			if !opts.Recursive {
				return "", fmt.Errorf("cannot remove '%s': Is a directory", path)
			}

			repoName := strings.TrimPrefix(targetPath, "/")
			delete(s.Repos, repoName) // Remove from Repos map

			// Remove from Filesystem
			err = s.RemoveAll(targetPath)
			if err != nil {
				return "", fmt.Errorf("failed to remove %s: %v", path, err)
			}
		} else {
			// File
			err = s.Filesystem.Remove(targetPath)
			if err != nil {
				return "", fmt.Errorf("failed to remove file %s: %v", path, err)
			}
		}
		removed = append(removed, path)
	}

	if len(removed) == 0 && !opts.Force {
		return "", fmt.Errorf("no files removed")
	}

	if len(removed) > 0 {
		return fmt.Sprintf("Removed %s", strings.Join(removed, ", ")), nil
	}
	return "", nil
}

func (c *RmCommand) Help() string {
	return `📘 RM (1)                                               Shell Manual

 💡 DESCRIPTION
    ・ファイルやフォルダを削除する（復元できません）
    
    ⚠️ 注意: これは ` + "`git rm`" + ` ではなく、シェルの ` + "`rm`" + ` コマンド相当です。
    インデックス（ステージングエリア）からの削除は行われません。
    追跡対象のファイルを削除した場合は、その後 ` + "`git add`" + ` で削除を記録する必要があります。

 📋 SYNOPSIS
    rm [-rf] <path>

 ⚙️  COMMON OPTIONS
    (暗黙的) -rf
        ディレクトリの場合は再帰的に、強制的に削除します。

 🛠  EXAMPLES
    1. ファイルを削除
       $ rm file.txt
    
    2. ディレクトリを削除
       $ rm dir/
`
}
