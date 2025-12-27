package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
	if opts.Ref1 == "" {
		// git diff (no args) -> worktree vs index (not supported fully in simulation yet?)
		// The original code returned usage message.
		// Standard git diff (no args) is Worktree vs Index.
		// git diff --cached is Index vs HEAD.
		// git diff A B is A vs B.
		if opts.Cached {
			// diff --cached (Index vs HEAD)
			// support later?
			return nil, fmt.Errorf("diff --cached not yet supported in simulation (requires Index inspection)")
		}
		return nil, fmt.Errorf("usage: git diff <ref1> <ref2>\n(Worktree diff not yet supported)")
	}

	if opts.Ref2 == "" {
		// diff A -> A vs Worktree? Or A vs HEAD?
		// git diff <commit> -> <commit> vs Worktree
		return nil, fmt.Errorf("usage: git diff <ref1> <ref2>\n(Single ref diff not yet supported)")
	}

	return opts, nil
}

func (c *DiffCommand) executeDiff(_ *git.Session, repo *gogit.Repository, opts *DiffOptions) (string, error) {
	// Resolve refs
	h1, err := repo.ResolveRevision(plumbing.Revision(opts.Ref1))
	if err != nil {
		return "", err
	}
	h2, err := repo.ResolveRevision(plumbing.Revision(opts.Ref2))
	if err != nil {
		return "", err
	}

	c1, err := repo.CommitObject(*h1)
	if err != nil {
		return "", err
	}
	c2, err := repo.CommitObject(*h2)
	if err != nil {
		return "", err
	}

	tree1, err := c1.Tree()
	if err != nil {
		return "", err
	}
	tree2, err := c2.Tree()
	if err != nil {
		return "", err
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
