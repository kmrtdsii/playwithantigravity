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

func (c *RmCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: rm [-rf] <path>")
	}

	// Parse arguments
	var paths []string
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue // skip flags like -rf
		}
		paths = append(paths, arg)
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("usage: rm [-rf] <path>...")
	}

	var removed []string
	for _, path := range paths {
		// Safety check: Don't allow deleting root or critical paths if possible
		if path == "/" || path == "." || path == ".." {
			continue // skip unsafe
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
			// If not found, just skip or error? strictly rm errors.
			// But for multiple args, usually it errors but might continue?
			// Let's error for now to be safe/simple, or just note it.
			continue
		}

		// Check if it is a directory representing a repo
		if fi.IsDir() {
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

	if len(removed) == 0 {
		return "", fmt.Errorf("no files removed")
	}

	return fmt.Sprintf("Removed %s", strings.Join(removed, ", ")), nil
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
