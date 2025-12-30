package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("show", func() git.Command { return &ShowCommand{} })
}

type ShowCommand struct{}

// Ensure ShowCommand implements git.Command
var _ git.Command = (*ShowCommand)(nil)

type ShowOptions struct {
	NameStatus bool
	CommitID   string
}

func (c *ShowCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
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

	return c.executeShow(s, repo, opts)
}

func (c *ShowCommand) parseArgs(args []string) (*ShowOptions, error) {
	opts := &ShowOptions{
		CommitID: "HEAD", // Default
	}
	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		if arg == "--name-status" {
			opts.NameStatus = true
		} else if strings.HasPrefix(arg, "--format=") {
			// ignore
		} else if arg == "-h" || arg == "--help" {
			return nil, fmt.Errorf("help requested")
		} else if !strings.HasPrefix(arg, "-") {
			opts.CommitID = arg
		}
	}
	return opts, nil
}

func (c *ShowCommand) executeShow(_ *git.Session, repo *gogit.Repository, opts *ShowOptions) (string, error) {
	h, err := repo.ResolveRevision(plumbing.Revision(opts.CommitID))
	if err != nil {
		// If revision lookup fails, try to treat it as a file path at HEAD
		// This supports 'git show README.md' -> 'git show HEAD:README.md'
		if err.Error() == "reference not found" {
			headRef, headErr := repo.Head()
			if headErr != nil {
				return "", fmt.Errorf("reference not found: %s", opts.CommitID)
			}

			headCommit, commitErr := repo.CommitObject(headRef.Hash())
			if commitErr != nil {
				return "", fmt.Errorf("reference not found: %s", opts.CommitID)
			}

			tree, treeErr := headCommit.Tree()
			if treeErr != nil {
				return "", fmt.Errorf("reference not found: %s", opts.CommitID)
			}

			// Try to find the file in the HEAD tree
			file, fileErr := tree.File(opts.CommitID)
			if fileErr == nil {
				// Found as file! Return content.
				content, err := file.Contents()
				if err != nil {
					return "", err
				}
				return content, nil
			}
		}

		// Map generic error to user friendly message
		if err.Error() == "reference not found" {
			return "", fmt.Errorf("fatal: ambiguous argument '%s': unknown revision or path not in the working tree.", opts.CommitID)
		}
		return "", err
	}

	commit, err := repo.CommitObject(*h)
	if err != nil {
		return "", err
	}

	if !opts.NameStatus {
		// Basic commit info + Patch
		var sb strings.Builder
		sb.WriteString(commit.String())
		sb.WriteString("\n")

		// Calculate Diff with Parent for Patch
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
		}

		currentTree, err := commit.Tree()
		if err != nil {
			return "", err
		}

		if parentTree != nil {
			patch, err := parentTree.Patch(currentTree)
			if err != nil {
				return "", err
			}
			sb.WriteString(patch.String())
		} else {
			// Root diff - everything is added
			// For root commit, Patch(nil) might not work as expected or verify behavior
			// go-git's Patch method expects two trees.
			// Emulate patch for root? Or just list files.
			// Standard git show shows diff /dev/null
			// Let's rely on name-status logic or just skip patch for root for now to be safe,
			// or better: iterate files and print content as + lines.
			// For simplicity and standard go-git limitations, we might just show header for root.
			// But let's try Patch with empty tree if possible.
			// emptyTree := &object.Tree{}
			// patch, _ := emptyTree.Patch(currentTree)
			// sb.WriteString(patch.String())
			// NOTE: Creating empty tree object cleanly is tricky without hashing.
			// Just skipping Patch for root to prevent crash, simple show is better than error.
		}

		return sb.String(), nil
	}

	// Calculate Diff with Parent
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
	}

	currentTree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	var changes object.Changes
	if parentTree != nil {
		changes, err = parentTree.Diff(currentTree)
	} else {
		// Root diff
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
			status = "M"
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
