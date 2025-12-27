package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("diff", func() git.Command { return &DiffCommand{} })
}

type DiffCommand struct{}

func (c *DiffCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse flags
	var refs []string

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-h", "--help":
			return c.Help(), nil
		default:
			if !strings.HasPrefix(arg, "-") {
				refs = append(refs, arg)
			}
		}
	}

	if len(refs) < 2 {
		return "usage: git diff <ref1> <ref2>\n(Worktree diff not yet supported)", nil
	}
	ref1 := refs[0]
	ref2 := refs[1]

	// Resolve refs
	h1, err := repo.ResolveRevision(plumbing.Revision(ref1))
	if err != nil {
		return "", err
	}
	h2, err := repo.ResolveRevision(plumbing.Revision(ref2))
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
