package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("diff", func() git.Command { return &DiffCommand{} })
}

type DiffCommand struct{}

// Ensure DiffCommand implements git.Command
var _ git.Command = (*DiffCommand)(nil)

type DiffOptions struct {
	Cached bool
	Ref1   string
	Ref2   string
}

func (c *DiffCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
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

	return c.executeDiff(s, repo, opts)
}

func (c *DiffCommand) parseArgs(args []string) (*DiffOptions, error) {
	opts := &DiffOptions{}
	var refs []string

	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		switch arg {
		case "--cached", "--staged":
			opts.Cached = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if !strings.HasPrefix(arg, "-") {
				refs = append(refs, arg)
			}
		}
	}

	if len(refs) > 0 {
		opts.Ref1 = refs[0]
	}
	if len(refs) > 1 {
		opts.Ref2 = refs[1]
	}

	// Validation
	// git diff -> Ref1="", Ref2="", Cached=false
	// git diff --cached -> Ref1="", Ref2="", Cached=true
	// git diff commit -> Ref1="commit", Ref2="", Cached=false

	return opts, nil
}

func (c *DiffCommand) executeDiff(s *git.Session, repo *gogit.Repository, opts *DiffOptions) (string, error) {
	var tree1, tree2 *object.Tree
	var err error

	// 1. Resolve Tree 2 (Target)
	if opts.Ref2 != "" {
		// git diff ref1 ref2
		h2, err := repo.ResolveRevision(plumbing.Revision(opts.Ref2))
		if err != nil {
			return "", fmt.Errorf("could not resolve %s: %w", opts.Ref2, err)
		}
		commit2, err := repo.CommitObject(*h2)
		if err != nil {
			return "", err
		}
		tree2, err = commit2.Tree()
		if err != nil {
			return "", err
		}
	} else if opts.Ref1 != "" && !opts.Cached {
		// git diff ref1 -> ref1 vs Worktree
		// Standard Git: git diff <commit> compares <commit> with working tree
		tree2, err = s.GetWorktreeTree(repo)
		if err != nil {
			return "", fmt.Errorf("failed to build worktree tree: %w", err)
		}
	} else if opts.Cached {
		// git diff --cached -> Index vs HEAD
		tree2, err = s.GetIndexTree(repo)
		if err != nil {
			return "", fmt.Errorf("failed to build index tree: %w", err)
		}
	} else {
		// git diff -> Worktree vs Index (or HEAD if no index?)
		// Standard Git: compares working directory with index
		tree2, err = s.GetWorktreeTree(repo)
		if err != nil {
			return "", fmt.Errorf("failed to build worktree tree: %w", err)
		}
	}

	// 2. Resolve Tree 1 (Base)
	if opts.Ref1 != "" {
		h1, err := repo.ResolveRevision(plumbing.Revision(opts.Ref1))
		if err != nil {
			return "", fmt.Errorf("could not resolve %s: %w", opts.Ref1, err)
		}
		commit1, err := repo.CommitObject(*h1)
		if err != nil {
			return "", err
		}
		tree1, err = commit1.Tree()
		if err != nil {
			return "", err
		}
	} else if opts.Cached {
		// git diff --cached -> Index vs HEAD. Base is HEAD.
		head, err := repo.Head()
		if err != nil {
			// No commits yet, compare with empty tree
			tree1 = &object.Tree{}
		} else {
			commit1, err := repo.CommitObject(head.Hash())
			if err != nil {
				return "", err
			}
			tree1, err = commit1.Tree()
			if err != nil {
				return "", err
			}
		}
	} else {
		// git diff -> Worktree vs Index. Base is Index.
		tree1, err = s.GetIndexTree(repo)
		if err != nil {
			// If index tree fails (e.g. empty repo), fallback to HEAD or empty
			head, err := repo.Head()
			if err != nil {
				tree1 = &object.Tree{}
			} else {
				commit1, _ := repo.CommitObject(head.Hash())
				tree1, _ = commit1.Tree()
			}
		}
	}

	if tree1 == nil || tree2 == nil {
		return "", fmt.Errorf("internal error: could not resolve trees for diff")
	}

	patch, err := tree1.Patch(tree2)
	if err != nil {
		return "", err
	}

	return patch.String(), nil
}

func (c *DiffCommand) Help() string {
	return `📘 GIT-DIFF (1)                                         Git Manual

 💡 DESCRIPTION
    ・2つのコミットを比較して、変更内容（差分）を表示する
    ・ファイルの中身が具体的にどう変わったかを確認する
    
    ⚠️ 現在のバージョンでは、ワーキングツリーとインデックスの差分（引数なしの diff）はサポートされていません。
    2つのコミットを指定して比較してください。

 📋 SYNOPSIS
    git diff <commit1> <commit2>

 🛠  EXAMPLES
    1. 2つのコミットを比較
       $ git diff HEAD~1 HEAD

    2. ブランチ間を比較
       $ git diff main develop

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-diff
`
}
