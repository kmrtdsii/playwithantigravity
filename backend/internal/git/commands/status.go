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
	git.RegisterCommand("status", func() git.Command { return &StatusCommand{} })
}

type StatusCommand struct{}

// Ensure StatusCommand implements git.Command
var _ git.Command = (*StatusCommand)(nil)

type StatusOptions struct {
	Short  bool
	Branch bool
}

func (c *StatusCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	return c.executeStatus(s, repo, opts)
}

func (c *StatusCommand) parseArgs(args []string) (*StatusOptions, error) {
	opts := &StatusOptions{}
	// status command doesn't have many flags in simulation yet, but prepare structure
	for _, arg := range args[1:] {
		switch arg {
		case "-s", "--short":
			opts.Short = true
		case "-b", "--branch":
			opts.Branch = true
		case "-sb", "-bs":
			opts.Short = true
			opts.Branch = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("error: unknown option `%s`", arg)
			}
		}
	}
	return opts, nil
}

func (c *StatusCommand) executeStatus(_ *git.Session, repo *gogit.Repository, opts *StatusOptions) (string, error) {
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	status, err := w.Status()
	if err != nil {
		return "", err
	}

	if opts.Short {
		return c.formatShortInfo(repo, status, opts.Branch)
	}

	return c.formatLongInfo(repo, status)
}

func (c *StatusCommand) formatLongInfo(repo *gogit.Repository, status gogit.Status) (string, error) {
	var sb strings.Builder

	// 1. Branch Info
	head, err := repo.Head()
	if err == nil {
		if head.Name().IsBranch() {
			sb.WriteString(fmt.Sprintf("On branch %s\n", head.Name().Short()))
		} else {
			sb.WriteString(fmt.Sprintf("HEAD detached at %s\n", head.Hash().String()[:7]))
		}
	} else {
		sb.WriteString("No commits yet\n")
	}

	// 2. Classify Files
	var staged, unstaged, untracked []string

	paths := make([]string, 0, len(status))
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		s := status[path]

		// Untracked
		if s.Staging == gogit.Untracked {
			untracked = append(untracked, path)
			continue
		}

		// Staged changes (Staging has something other than Unmodified/Untracked)
		// Note: A file can be both queued for commit AND modified (staged + unstaged changes)
		if s.Staging != gogit.Unmodified && s.Staging != gogit.Untracked {
			staged = append(staged, fmt.Sprintf("%-12s%s", mapStatus(s.Staging), path))
		}

		// Unstaged changes (Worktree has something other than Unmodified)
		if s.Worktree != gogit.Unmodified && s.Worktree != gogit.Untracked {
			unstaged = append(unstaged, fmt.Sprintf("%-12s%s", mapStatus(s.Worktree), path))
		}
	}

	hasChanges := false

	// 3. Print Staged
	if len(staged) > 0 {
		sb.WriteString("\nChanges to be committed:\n  (use \"git restore --staged <file>...\" to unstage)\n")
		for _, line := range staged {
			sb.WriteString(fmt.Sprintf("\t\x1b[32m%s\x1b[0m\n", line)) // Green
		}
		hasChanges = true
	}

	// 4. Print Unstaged
	if len(unstaged) > 0 {
		sb.WriteString("\nChanges not staged for commit:\n  (use \"git add <file>...\" to update what will be committed)\n  (use \"git restore <file>...\" to discard changes in working directory)\n")
		for _, line := range unstaged {
			sb.WriteString(fmt.Sprintf("\t\x1b[31m%s\x1b[0m\n", line)) // Red
		}
		hasChanges = true
	}

	// 5. Print Untracked
	if len(untracked) > 0 {
		sb.WriteString("\nUntracked files:\n  (use \"git add <file>...\" to include in what will be committed)\n")
		for _, line := range untracked {
			sb.WriteString(fmt.Sprintf("\t\x1b[31m%s\x1b[0m\n", line)) // Red
		}
		hasChanges = true
	}

	if !hasChanges {
		sb.WriteString("nothing to commit, working tree clean\n")
	}

	return sb.String(), nil
}

func mapStatus(s gogit.StatusCode) string {
	switch s {
	case gogit.Modified:
		return "modified:"
	case gogit.Added:
		return "new file:"
	case gogit.Deleted:
		return "deleted:"
	case gogit.Renamed:
		return "renamed:"
	case gogit.Copied:
		return "copied:"
	case gogit.UpdatedButUnmerged:
		return "unmerged:"
	default:
		return string(s)
	}
}

func (c *StatusCommand) formatShortInfo(repo *gogit.Repository, status gogit.Status, showBranch bool) (string, error) {
	var sb strings.Builder

	if showBranch {
		head, err := repo.Head()
		if err == nil {
			if head.Name().IsBranch() {
				sb.WriteString(fmt.Sprintf("## %s\n", head.Name().Short()))
			} else {
				sb.WriteString(fmt.Sprintf("## HEAD (detached at %s)\n", head.Hash().String()[:7]))
			}
		} else {
			sb.WriteString("## No commits yet\n")
		}
	}

	// Sort paths for deterministic output
	var paths []string
	for path := range status {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		s := status[path]
		if s.Staging == gogit.Unmodified && s.Worktree == gogit.Unmodified {
			continue
		}

		// X (Staging status), Y (Worktree status)
		var x, y byte

		if s.Staging == gogit.Untracked {
			x = '?'
			y = '?'
		} else {
			x = getStatusCodeChar(s.Staging)
			y = getStatusCodeChar(s.Worktree)
		}

		sb.WriteString(fmt.Sprintf("%c%c %s\n", x, y, path))
	}

	return sb.String(), nil
}

func getStatusCodeChar(c gogit.StatusCode) byte {
	switch c {
	case gogit.Modified:
		return 'M'
	case gogit.Added:
		return 'A'
	case gogit.Deleted:
		return 'D'
	case gogit.Renamed:
		return 'R'
	case gogit.Copied:
		return 'C'
	case gogit.UpdatedButUnmerged:
		return 'U'
	case gogit.Untracked:
		return '?'
	default:
		// gogit.Unmodified (' ') or 0 or anything else -> Space
		return ' '
	}
}

func (c *StatusCommand) Help() string {
	return `📘 GIT-STATUS (1)                                       Git Manual

 💡 DESCRIPTION
    ・「どのファイルが変更されたか」を確認する
    ・「どのファイルがコミット準備できているか」を確認する
    ・現在のブランチや状況を確認する
    困ったら、まずこれを打つのが基本です。

 📋 SYNOPSIS
    git status [-s|--short] [-b|--branch]

 ⚙️  COMMON OPTIONS
    -s, --short
        変更ファイルだけを簡易表示します。
    -b, --branch
        ショート形式(-s)の際にもブランチ情報を表示します。
        （通常表示ではデフォルトで表示されるため、主に -s と組み合わせて使用します）

 🛠  PRACTICAL EXAMPLES
    1. 基本: 現状を確認する
       手が止まったらとりあえず打って、状況を把握します。
       $ git status

    2. 実践: 情報を絞って見る (Recommended)
       ブランチ名と変更ファイルだけを1行ずつ簡易表示します。
       情報量が絞られて見やすいため、現場ではこれをエイリアス(stなど)に登録して多用します。
       $ git status -sb

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-status
`
}
