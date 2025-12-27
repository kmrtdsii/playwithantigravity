package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("show", func() git.Command { return &ShowCommand{} })
}

type ShowCommand struct{}

func (c *ShowCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Basic parsing: git show --name-status <commit>
	// We ignore --format="" for now or just handle the commit arg.
	if len(args) < 2 {
		return "", fmt.Errorf("usage: git show <commit> [--name-status]")
	}

	// Simple arg parsing
	var commitID string
	showNameStatus := false

	for _, arg := range args[1:] {
		if arg == "--name-status" {
			showNameStatus = true
		} else if strings.HasPrefix(arg, "--format=") {
			// ignore
		} else if !strings.HasPrefix(arg, "-") {
			commitID = arg
		}
	}

	if commitID == "" {
		// Default to HEAD if not provided? git defaults to HEAD.
		commitID = "HEAD"
	}

	h, err := repo.ResolveRevision(plumbing.Revision(commitID))
	if err != nil {
		return "", err
	}

	commit, err := repo.CommitObject(*h)
	if err != nil {
		return "", err
	}

	if !showNameStatus {
		// Fallback to basic commit info (not implemented strictly, but enough for now)
		return commit.String(), nil
	}

	// Calculate Diff with Parent
	// If no parent (root), diff with empty tree
	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return "", err
		}
		parentTree, err = parent.Tree()
		if err != nil {
			return "", err
		}
	} else {
		// Root commit: Diff against empty tree?
		// go-git doesn't easily support "empty tree" object without construction?
		// We can just iterate the current tree and mark all as Added.
		// For simplicity, let's treat it as empty parent.
	}

	currentTree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	var changes object.Changes
	if parentTree != nil {
		changes, err = parentTree.Diff(currentTree)
	} else {
		// Root commit diff logic
		// Just listing all files as Added
		// go-git Diff might not handle nil parentTree well.
		// Workaround: custom walker or just accept empty changes for root for now.
		// Correct way: use object.DiffTree with emtpy tree hash?
		// Empty tree hash is usually 4b825dc642cb6eb9a060e54bf8d69288fbee4904
		// But let's verify if Diff handles nil.
		// It expects *Tree.
		return listRootChanges(currentTree)
	}

	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue
		}

		var status string
		var path string

		switch action {
		case merkletrie.Insert:
			status = "A"
			path = change.To.Name
		case merkletrie.Delete:
			status = "D"
			path = change.From.Name
		case merkletrie.Modify:
			status = "M"
			path = change.To.Name
		default:
			status = "M" // Rename or other?
			path = change.To.Name
		}

		sb.WriteString(fmt.Sprintf("%s\t%s\n", status, path))
	}

	return sb.String(), nil
}

func (c *ShowCommand) Help() string {
	return `📘 GIT-SHOW (1)                                         Git Manual

 💡 DESCRIPTION
    ・特定のコミットの変更内容やメッセージを詳しく表示する
    ・コミットの内容を詳細に確認する

 📋 SYNOPSIS
    git show [<commit>] [--name-status]

 ⚙️  COMMON OPTIONS
    --name-status
        変更内容の差分テキストではなく、変更されたファイル名と状態（A/M/D）のみを表示します。

 🛠  EXAMPLES
    1. 最新のコミットを表示
       $ git show

    2. 特定のコミットの変更ファイル一覧を表示
       $ git show --name-status e5a3b21

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-show
`
}

func listRootChanges(tree *object.Tree) (string, error) {
	var sb strings.Builder
	err := tree.Files().ForEach(func(f *object.File) error {
		sb.WriteString(fmt.Sprintf("A\t%s\n", f.Name))
		return nil
	})
	return sb.String(), err
}
