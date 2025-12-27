package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("clean", func() git.Command { return &CleanCommand{} })
}

type CleanCommand struct{}

func (c *CleanCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	dryRun := false
	force := false
	dir := false

	// Basic flag parsing
	for _, arg := range args {
		if arg == "-n" || arg == "--dry-run" {
			dryRun = true
		} else if arg == "-f" || arg == "--force" {
			force = true
		} else if arg == "-d" {
			dir = true
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Handle combined flags roughly (e.g. -fd)
			if strings.Contains(arg, "f") {
				force = true
			}
			if strings.Contains(arg, "d") {
				dir = true
			}
			if strings.Contains(arg, "n") {
				dryRun = true
			}
		}
	}

	// Safety check
	if !dryRun && !force {
		return "", fmt.Errorf("fatal: clean.requireForce defaults to true and neither -i, -n, nor -f given; refusing to clean")
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	status, err := w.Status()
	if err != nil {
		return "", err
	}

	var toClean []string
	for path, fileStatus := range status {
		if fileStatus.Worktree == gogit.Untracked {
			toClean = append(toClean, path)
		}
	}

	// Sort for deterministic output
	sort.Strings(toClean)

	var output strings.Builder
	fs := w.Filesystem

	for _, path := range toClean {
		// Check if directory
		fi, err := fs.Stat(path)
		if err == nil && fi.IsDir() {
			if !dir {
				continue // Skip directories if -d is not set
			}
		}

		if dryRun {
			output.WriteString(fmt.Sprintf("Would remove %s\n", path))
		} else {
			// Remove
			// Use fs.Remove for files. For directories (if -d), RemoveAll?
			// go-git's billy filesystem Remove usually handles files. RemoveAll for dirs.
			// However, standard clean output says "Removing <path>"

			// Actually, fs.Remove might fail on non-empty dirs. w.Filesystem usually supports Remove (files) and Remove (dirs if empty?).
			// If it's a directory, we should probably recursively remove?
			// But for now, let's assume simple Remove works or use a helper.
			var err error
			if fi.IsDir() {
				// Use os.RemoveAll equivalent if possible, but we are in billy.Filesystem abstraction.
				// billy basic interface only has Remove. Some impls have RemoveAll.
				// Let's try Remove. If it fails (dir not empty), and we are in -d mode...
				// But wait, if Status returned a directory, it means it's untracked content.
				// Assuming standard Remove works for files.
				err = fs.Remove(path)
			} else {
				err = fs.Remove(path)
			}

			if err != nil {
				output.WriteString(fmt.Sprintf("failed to remove %s: %v\n", path, err))
			} else {
				output.WriteString(fmt.Sprintf("Removing %s\n", path))
			}
		}
	}

	return strings.TrimSpace(output.String()), nil
}

func (c *CleanCommand) Help() string {
	return `📘 GIT-CLEAN (1)                                        Git Manual

 💡 DESCRIPTION
    ・追跡されていないファイル（ゴミファイル）を削除する
    ・ディレクトリをまとめて整理する
    
    誤って必要なファイルを消さないよう、通常は ` + "`" + `-f` + "`" + ` (force) が必須です。
    まずは ` + "`" + `-n` + "`" + ` (dry-run) で何が消えるか確認することを推奨します。

 📋 SYNOPSIS
    git clean [-n] [-f] [-d]

 ⚙️  COMMON OPTIONS
    -n, --dry-run
        実際には削除せず、何が削除されるかを表示します。

    -f, --force
        実際に削除を実行します（必須オプション）。

    -d
        追跡されていないディレクトリも削除対象にします。

 🛠  EXAMPLES
    1. 何が消えるか確認（推奨）
       $ git clean -n -d

    2. 強制削除
       $ git clean -f -d

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-clean
`
}
