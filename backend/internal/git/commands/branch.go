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

func (c *BranchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Basic Argument Parsing
	// supported flags: -d, -D, -m, -r, -a, -f, --help
	var (
		deleteMode bool
		force      bool // -D or -f depending on context
		moveMode   bool
		remoteMode bool
		allMode    bool
		helpMode   bool
		branchName string
		secondArg  string // newBranchName for move, startPoint for create
	)

	// Skip the first arg which is "branch"
	cmdArgs := args[1:]

	// If no arguments, it's a list command
	if len(cmdArgs) == 0 {
		return c.listBranches(s.GetRepo(), false, false)
	}

	// Parse flags manually to handle mixed order if needed
	for i := 0; i < len(cmdArgs); i++ {
		arg := strings.TrimSpace(cmdArgs[i])
		switch arg {
		case "--help", "-h":
			helpMode = true
		case "-d", "--delete":
			deleteMode = true
		case "-D":
			deleteMode = true
			force = true
		case "-m", "--move":
			moveMode = true
		case "-f", "--force":
			force = true
		case "-r", "--remotes":
			remoteMode = true
		case "-a", "--all":
			allMode = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", fmt.Errorf("unknown option: %s", arg)
			}
			if branchName == "" {
				branchName = arg
			} else if secondArg == "" {
				secondArg = arg
			}
		}
	}

	if helpMode {
		return c.Help(), nil
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	// 2. Dispatch based on mode

	// LIST
	if !deleteMode && !moveMode {
		// Use explicit list flag check if we had one, but strict check:
		// git branch <name> -> creation
		// git branch -> list
		// git branch -r -> list
		// git branch -a -> list

		if branchName == "" {
			return c.listBranches(repo, remoteMode, allMode)
		}
		// Special case: git branch -r <name> is technically list pattern matching in git, but here likely means list?
		// But usually creation doesn't use -r / -a.
		if remoteMode || allMode {
			return c.listBranches(repo, remoteMode, allMode)
		}

		// Creation
		startPoint := "HEAD"
		if secondArg != "" {
			startPoint = secondArg
		}

		return c.createBranch(repo, branchName, startPoint, force)
	}

	// DELETE
	if deleteMode {
		if branchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		return c.deleteBranch(repo, branchName, force, remoteMode)
	}

	// MOVE
	if moveMode {
		if branchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		if secondArg == "" {
			// Rename current branch
			head, err := repo.Head()
			if err != nil {
				return "", fmt.Errorf("cannot rename current branch: HEAD invalid")
			}
			if !head.Name().IsBranch() {
				return "", fmt.Errorf("cannot rename detached HEAD")
			}
			secondArg = branchName
			branchName = head.Name().Short()
		}
		return c.moveBranch(repo, branchName, secondArg, force)
	}

	return "", nil
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
		rs, err := repo.Remotes()
		if err == nil {
			for _, r := range rs {
				refs, listErr := r.List(&gogit.ListOptions{}) // basic list
				if listErr == nil {
					for _, ref := range refs {
						if ref.Name().IsRemote() {
							// strip refs/remotes/
							name := ref.Name().Short()
							// Short() often gives origin/master for refs/remotes/origin/master
							branches = append(branches, name)
						}
					}
				}
			}
			// Fallback: iterate all references and filter
			refs, _ := repo.References()
			_ = refs.ForEach(func(r *plumbing.Reference) error {
				// if r.Name().IsRemote() {
				// 	// branches = append(branches, r.Name().Short())
				// }
				return nil
			})
		}
		// Actually go-git `repo.References()` contains remotes too.
		// Let's just use References() and filter.
		refs, err := repo.References()
		if err != nil {
			return "", err
		}
		_ = refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsRemote() {
				// Only add if we are in remote/all mode
				// Avoid duplicates if possible, but for now simple list
				exists := false
				short := r.Name().Short()
				for _, b := range branches {
					if b == short {
						exists = true
						break
					}
				}
				if !exists {
					branches = append(branches, short)
				}
			}
			return nil
		})
	}

	return strings.Join(branches, "\n"), nil
}

func (c *BranchCommand) createBranch(repo *gogit.Repository, name, startPoint string, force bool) (string, error) {
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("unknown switch configuration: %s", name)
	}

	hash, err := git.ResolveRevision(repo, startPoint)
	if err != nil {
		return "", fmt.Errorf("not a valid object name: '%s'", startPoint)
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

		if !force {
			return "", fmt.Errorf("fatal: A branch named '%s' already exists.", name)
		}
		// If force is true, we proceed to overwrite
	}

	// Create or Overwrite reference
	newRef := plumbing.NewHashReference(refName, *hash)

	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	// If overwritten, message might differ? Git usually silent or "Reset branch..."?
	// But "Created branch" is simple for now.
	return "Created branch " + name, nil
}

func (c *BranchCommand) deleteBranch(repo *gogit.Repository, name string, force, remote bool) (string, error) {
	// TODO: support remote delete (git branch -dr origin/branch)
	if remote {
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

func (c *BranchCommand) moveBranch(repo *gogit.Repository, oldName, newName string, force bool) (string, error) {
	oldRefName := plumbing.ReferenceName("refs/heads/" + oldName)
	oldRef, err := repo.Reference(oldRefName, true)
	if err != nil {
		return "", fmt.Errorf("branch '%s' not found", oldName)
	}

	newRefName := plumbing.ReferenceName("refs/heads/" + newName)
	// check if exists
	_, err = repo.Reference(newRefName, true)
	if err == nil && !force {
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
