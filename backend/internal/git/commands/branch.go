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
	git.RegisterCommand("branch", func() git.Command { return &BranchCommand{} })
}

type BranchCommand struct{}

// Ensure BranchCommand implements git.Command
var _ git.Command = (*BranchCommand)(nil)

type BranchOptions struct {
	Delete      bool
	DeleteForce bool
	Move        bool
	StartPoint  string
	BranchName  string
	NewName     string
	Remote      bool
	All         bool
	Force       bool
}

func (c *BranchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Parse Args
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

	// 2. Dispatch
	// LIST
	if !opts.Delete && !opts.DeleteForce && !opts.Move {
		if opts.BranchName == "" {
			return c.listBranches(repo, opts.Remote, opts.All)
		}
		// Special case: "git branch -r" or "git branch -a" without name is list
		if opts.Remote && !opts.Move && !opts.Delete { // "git branch -r"
			return c.listBranches(repo, opts.Remote, opts.All)
		}
		if opts.All && !opts.Move && !opts.Delete { // "git branch -a"
			return c.listBranches(repo, opts.Remote, opts.All)
		}

		// If name provided but not Delete/Move, it's CREATE
		return c.createBranch(repo, opts)
	}

	// DELETE
	if opts.Delete || opts.DeleteForce {
		if opts.BranchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		return c.deleteBranch(repo, opts)
	}

	// MOVE
	if opts.Move {
		if opts.BranchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		return c.moveBranch(repo, opts)
	}

	return "", nil
}

func (c *BranchCommand) parseArgs(args []string) (*BranchOptions, error) {
	opts := &BranchOptions{
		StartPoint: "HEAD",
	}
	cmdArgs := args[1:]

	// Collect arguments to determine Name and StartPoint/NewName
	var cleanArgs []string

	for _, arg := range cmdArgs {
		switch arg {
		case "--help", "-h":
			return nil, fmt.Errorf("help requested")
		case "-d", "--delete":
			opts.Delete = true
		case "-D":
			opts.DeleteForce = true // Implies Force for deletion logic
		case "-m", "--move":
			opts.Move = true
		case "-f", "--force":
			opts.Force = true
		case "-r", "--remotes":
			opts.Remote = true
		case "-a", "--all":
			opts.All = true
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown option: %s", arg)
			}
			cleanArgs = append(cleanArgs, arg)
		}
	}

	if len(cleanArgs) > 0 {
		opts.BranchName = cleanArgs[0]
	}
	if len(cleanArgs) > 1 {
		if opts.Move {
			opts.NewName = cleanArgs[1]
		} else {
			opts.StartPoint = cleanArgs[1]
		}
	}

	// Handle Rename Current Branch: "git branch -m newname"
	// Here cleanArgs[0] is newname if we are renaming CURRENT.
	// Logic inside moveBranch needs to handle implicit "current" if only 1 arg.
	// Actually, if len==1 and Move==true, cleanArgs[0] *is* NewName, and BranchName (old) is implicit.
	// Let's refine parsing for Move case:
	if opts.Move && len(cleanArgs) == 1 {
		opts.NewName = cleanArgs[0]
		opts.BranchName = "" // Signal to resolve current
	} else if opts.Move && len(cleanArgs) >= 2 {
		opts.BranchName = cleanArgs[0] // Old
		opts.NewName = cleanArgs[1]    // New
	}

	return opts, nil
}

func (c *BranchCommand) listBranches(repo *gogit.Repository, remote, all bool) (string, error) {
	// Collect branches
	var branches []string

	// Local branches
	if !remote || all {
		bs, err := repo.Branches()
		if err != nil {
			return "", err
		}
		_ = bs.ForEach(func(r *plumbing.Reference) error {
			branches = append(branches, r.Name().Short())
			return nil
		})
	}

	// Remote branches
	if remote || all {
		remotes, err := c.listRemoteBranches(repo)
		if err != nil {
			return "", err
		}
		// Merge specific logic: deduplicate against existing local branches?
		// The original logic was appending to 'branches'.
		// Let's verify duplication handling.
		// Original logic:
		// Check duplicates against 'branches' (which contains local branches if 'all' is true)
		for _, rBranch := range remotes {
			exists := false
			for _, b := range branches {
				if b == rBranch {
					exists = true
					break
				}
			}
			if !exists {
				branches = append(branches, rBranch)
			}
		}
	}

	return strings.Join(branches, "\n"), nil
}

func (c *BranchCommand) createBranch(repo *gogit.Repository, opts *BranchOptions) (string, error) {
	name := opts.BranchName

	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("unknown switch configuration: %s", name)
	}

	hash, err := git.ResolveRevision(repo, opts.StartPoint)
	if err != nil {
		return "", fmt.Errorf("not a valid object name: '%s'", opts.StartPoint)
	}

	refName := plumbing.ReferenceName("refs/heads/" + name)

	// Check if branch already exists
	existingRef, err := repo.Storer.Reference(refName)
	if err == nil && existingRef != nil {
		// Existing logic
		head, headErr := repo.Head()
		if headErr == nil && head.Name() == refName {
			return "", fmt.Errorf("fatal: Cannot force update the current branch.")
		}

		if !opts.Force {
			return "", fmt.Errorf("fatal: A branch named '%s' already exists.", name)
		}
		// If force is true, we proceed to overwrite
	}

	// Create or Overwrite reference
	newRef := plumbing.NewHashReference(refName, *hash)

	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	return "Created branch " + name, nil
}

func (c *BranchCommand) deleteBranch(repo *gogit.Repository, opts *BranchOptions) (string, error) {
	name := opts.BranchName
	// TODO: support remote delete (git branch -dr origin/branch)
	if opts.Remote {
		return "", fmt.Errorf("deleting remote-tracking branches not fully supported yet in simulation")
	}

	refName := plumbing.ReferenceName("refs/heads/" + name)
	targetRef, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("branch '%s' not found", name)
	}

	// Prevent deleting current branch if not forced? Git prevents it always unless detached.
	headRef, err := repo.Head()
	if err == nil && headRef.Name() == refName {
		return "", fmt.Errorf("cannot delete branch '%s' checked out at current worktree", name)
	}

	// Determine if Force is needed (DeleteForce or just force flag logic?)
	// git branch -d checks merge. git branch -D skips check.
	force := opts.DeleteForce

	if !force {
		// Check if fully merged into HEAD
		// We need to check if branch (targetRef.Hash) is ancestor of HEAD (headRef.Hash)
		// IsFastForward(repo, base, target) -> returns true if base is ancestor of target
		// So IsFastForward(repo, targetRef.Hash, headRef.Hash)

		isMerged, err := git.IsFastForward(repo, targetRef.Hash(), headRef.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to check merge status: %w", err)
		}

		if !isMerged {
			return "", fmt.Errorf("the branch '%s' is not fully merged.\nIf you are sure you want to delete it, run 'git branch -D %s'", name, name)
		}
	}

	if err := repo.Storer.RemoveReference(refName); err != nil {
		return "", err
	}
	return "Deleted branch " + name, nil
}

func (c *BranchCommand) moveBranch(repo *gogit.Repository, opts *BranchOptions) (string, error) {
	oldName := opts.BranchName
	newName := opts.NewName

	oldRefName := plumbing.ReferenceName("refs/heads/" + oldName)
	oldRef, err := repo.Reference(oldRefName, true)
	if err != nil {
		return "", fmt.Errorf("branch '%s' not found", oldName)
	}

	newRefName := plumbing.ReferenceName("refs/heads/" + newName)
	// check if exists
	_, err = repo.Reference(newRefName, true)
	if err == nil && !opts.Force {
		return "", fmt.Errorf("branch '%s' already exists", newName)
	}

	// Rename: create new, delete old
	newRef := plumbing.NewHashReference(newRefName, oldRef.Hash())
	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}
	if err := repo.Storer.RemoveReference(oldRefName); err != nil {
		return "", err // inconsistent state risk, but simulation
	}

	return fmt.Sprintf("Renamed branch %s to %s", oldName, newName), nil
}

func (c *BranchCommand) listRemoteBranches(repo *gogit.Repository) ([]string, error) {
	var remoteBranches []string
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}
	_ = refs.ForEach(func(r *plumbing.Reference) error {
		if r.Name().IsRemote() {
			short := r.Name().Short()
			// Basic deduplication within remote list itself?
			// The caller deduplicates against local.
			remoteBranches = append(remoteBranches, short)
		}
		return nil
	})
	return remoteBranches, nil
}

func (c *BranchCommand) Help() string {
	return `📘 GIT-BRANCH (1)                                       Git Manual

 💡 DESCRIPTION
    ブランチ（作業の分岐）に関する以下の操作を行います：
    ・ブランチの一覧を表示する（引数なし）
    ・新しいブランチを作成する
    ・ブランチ名を変更する（-m）
    ・不要なブランチを削除する（-d）

 📋 SYNOPSIS
    git branch [--list] [-a] [-r]
    git branch [-f] <branchname> [<start-point>]
    git branch -d|-D <branchname>
    git branch -m <old> <new>

 ⚙️  COMMON OPTIONS
    -a, --all
        ローカルとリモート（追跡）の両方のブランチを表示します。

    -r, --remotes
        リモートブランチのみを表示します。

    -f, --force
        ブランチ作成時、同名のブランチが既に存在していても強制的に上書き（リセット）します。

    -d
        ブランチを削除します（マージ済みの安全な場合のみ）。

    -D
        ブランチを強制削除します（マージされていなくても削除）。

    -m
        ブランチ名を変更（移動）します。

    <start-point>
        新しいブランチの作成元となるコミットやブランチを指定します（デフォルトはHEAD）。

 🛠  EXAMPLES
    1. ブランチ一覧を表示
       $ git branch

    2. 新しいブランチを作成
       $ git branch feature/login

    3. 特定のコミットからブランチを作成
       $ git branch feature/fix-v1 e5a3b21

    4. 既存のブランチを強制上書き
       $ git branch -f existing-branch HEAD~1

    5. ブランチを強制削除
       $ git branch -D old-feature

    6. ブランチ名を変更
       $ git branch -m old-name new-name
       $ git branch -m new-name （現在のブランチ名を変更）
`
}
