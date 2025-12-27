package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func (c *CleanCommand) executeClean(s *git.Session, repo *gogit.Repository, opts *CleanOptions) (string, error) {
	if !opts.Force && !opts.DryRun {
		return "", fmt.Errorf("fatal: clean.requireForce defaults to true and neither -i, -n, nor -f given; refusing to clean")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	status, err := w.Status()
	if err != nil {
		return "", err
	}

	var candidates []string
	for path, fStatus := range status {
		if fStatus.Worktree == gogit.Untracked {
			candidates = append(candidates, path)
		}
	}

	fs := w.Filesystem
	var toRemoveFiles []string
	uniqueDirs := make(map[string]bool)

	for _, path := range candidates {
		info, err := fs.Lstat(path)
		if err != nil {
			fmt.Printf("DEBUG: Lstat failed for %s: %v\n", path, err)
			continue
		}

		fmt.Printf("DEBUG: processing %s IsDir=%v Dir=%v\n", path, info.IsDir(), opts.Dir)

		if info.IsDir() {
			if opts.Dir {
				// If Status returned a directory, we remove it directly
				// But we also might want to remove its parents?
				uniqueDirs[path] = true
			}
		} else {
			// File
			toRemoveFiles = append(toRemoveFiles, path)
			if opts.Dir {
				// Collect parent directories
				dir := filepath.Dir(path)
				for dir != "." && dir != "/" && dir != "\\" {
					uniqueDirs[dir] = true
					dir = filepath.Dir(dir)
				}
			}
		}
	}

	// Remove files First
	sort.Strings(toRemoveFiles) // Deterministic order

	var sb strings.Builder
	for _, path := range toRemoveFiles {
		prefix := "Removing"
		if opts.DryRun {
			prefix = "Would remove"
		} else {
			err := fs.Remove(path)
			if err != nil {
				return "", fmt.Errorf("failed to remove %s: %v", path, err)
			}
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, path))
	}

	// Remove Directories if opts.Dir
	if opts.Dir {
		var toRemoveDirs []string
		for dir := range uniqueDirs {
			toRemoveDirs = append(toRemoveDirs, dir)
		}
		// Sort descending by length to remove children before parents
		sort.Slice(toRemoveDirs, func(i, j int) bool {
			return len(toRemoveDirs[i]) > len(toRemoveDirs[j])
		})

		fmt.Printf("DEBUG: uniqueDirs=%v toRemoveDirs=%v\n", uniqueDirs, toRemoveDirs)

		for _, dir := range toRemoveDirs {
			// We only remove if empty (implied by fs.Remove on dir)
			// But we shouldn't fail if not empty (it means it had tracked files or we couldn't remove all children)

			// Check if exists first (might have been removed if nested)
			_, err := fs.Lstat(dir)
			if err != nil {
				continue
			}

			prefix := "Removing"
			if opts.DryRun {
				prefix = "Would remove"
			} else {
				err := fs.Remove(dir)
				if err != nil {
					// Report error for debugging
					sb.WriteString(fmt.Sprintf("DEBUG: failed to remove dir %s: %v\n", dir, err))
					continue
				}
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, dir))
		}
	} else {
		fmt.Println("DEBUG: opts.Dir is false")
	}

	return sb.String(), nil
}

func init() {
	git.RegisterCommand("clean", func() git.Command { return &CleanCommand{} })
}

type CleanCommand struct{}

type CleanOptions struct {
	DryRun bool
	Force  bool
	Dir    bool
	Args   []string
}

func (c *CleanCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	return c.executeClean(s, repo, opts)
}

func (c *CleanCommand) parseArgs(args []string) (*CleanOptions, error) {
	opts := &CleanOptions{}
	cmdArgs := args[1:]

	for _, arg := range cmdArgs {
		if arg == "-n" || arg == "--dry-run" {
			opts.DryRun = true
		} else if arg == "-f" || arg == "--force" {
			opts.Force = true
		} else if arg == "-d" {
			opts.Dir = true
		} else if arg == "-h" || arg == "--help" {
			return nil, fmt.Errorf("help requested")
		} else if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			// Combined short flags
			for _, char := range arg[1:] {
				switch char {
				case 'n':
					opts.DryRun = true
				case 'f':
					opts.Force = true
				case 'd':
					opts.Dir = true
				default:
					return nil, fmt.Errorf("unknown flag: -%c", char)
				}
			}
		} else {
			opts.Args = append(opts.Args, arg)
		}
	}
	return opts, nil
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
