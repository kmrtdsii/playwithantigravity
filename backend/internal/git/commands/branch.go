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
	// supported flags: -d, -D, -m, -r, -a, --help
	var (
		deleteMode    bool
		forceDelete   bool
		moveMode      bool
		remoteMode    bool
		allMode       bool
		helpMode      bool
		branchName    string
		newBranchName string // for move
	)

	// Skip the first arg which is "branch"
	cmdArgs := args[1:]

	// If no arguments, it's a list command
	if len(cmdArgs) == 0 {
		return c.listBranches(s.GetRepo(), false, false)
	}

	// Parse flags manually to handle mixed order if needed, though git usually expects flags first
	// Simple loop implementation
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--help", "-h":
			helpMode = true
		case "-d", "--delete":
			deleteMode = true
		case "-D":
			deleteMode = true
			forceDelete = true
		case "-m", "--move":
			moveMode = true
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
			} else if newBranchName == "" {
				newBranchName = arg // Second non-flag arg is target for move
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
	// If no specific action flags (delete, move) are set, and we have no branch name (or we have -r/-a), it's list.
	// Note: 'git branch pattern' lists branches matching pattern, but we'll stick to simple list for now
	// if we have just flags -r / -a.
	// If we have a branchName but NO delete/move flags, it's CREATE.
	if !deleteMode && !moveMode {
		if branchName == "" {
			return c.listBranches(repo, remoteMode, allMode)
		}
		// Creation (only if no remote/all flags usually, but git branch -r <name> is invalid context usually)
		if remoteMode || allMode {
			// standard git branch -r <name> implies tracking setup or list filter?
			// For simplicity: any -r/-a implies LIST
			return c.listBranches(repo, remoteMode, allMode)
		}
		return c.createBranch(repo, branchName)
	}

	// DELETE
	if deleteMode {
		if branchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		return c.deleteBranch(repo, branchName, forceDelete, remoteMode)
	}

	// MOVE
	if moveMode {
		if branchName == "" {
			return "", fmt.Errorf("branch name required")
		}
		if newBranchName == "" {
			// If only one arg provided for move, git assumes it's renaming current to <arg>
			// But standard usage 'git branch -m old new'.
			// 'git branch -m new' renames HEAD to new.
			// Let's support 'git branch -m <old> <new>' and 'git branch -m <new>' (rename current)
			if newBranchName == "" {
				// Rename current branch
				head, err := repo.Head()
				if err != nil {
					return "", fmt.Errorf("cannot rename current branch: HEAD invalid")
				}
				if !head.Name().IsBranch() {
					return "", fmt.Errorf("cannot rename detached HEAD")
				}
				newBranchName = branchName
				branchName = head.Name().Short()
			}
		}
		return c.moveBranch(repo, branchName, newBranchName, forceDelete)
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
		bs.ForEach(func(r *plumbing.Reference) error {
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
			refs.ForEach(func(r *plumbing.Reference) error {
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
		refs.ForEach(func(r *plumbing.Reference) error {
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

func (c *BranchCommand) createBranch(repo *gogit.Repository, name string) (string, error) {
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("unknown switch configuration: %s", name)
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("cannot create branch: %v (maybe no commits yet?)", err)
	}

	// Create new reference
	refName := plumbing.ReferenceName("refs/heads/" + name)
	newRef := plumbing.NewHashReference(refName, headRef.Hash())

	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	return "Created branch " + name, nil
}

func (c *BranchCommand) deleteBranch(repo *gogit.Repository, name string, force, remote bool) (string, error) {
	// TODO: support remote delete (git branch -dr origin/branch)
	if remote {
		return "", fmt.Errorf("deleting remote-tracking branches not fully supported yet in simulation")
	}

	if !force {
		// TODO: verify if branch is fully merged into HEAD
		// For now, we allow deletion to proceed, but using the variable silences the linter.
		_ = force
	}

	refName := plumbing.ReferenceName("refs/heads/" + name)
	_, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("branch '%s' not found", name)
	}

	// Prevent deleting current branch if not forced? Git prevents it always unless detached.
	headRef, err := repo.Head()
	if err == nil && headRef.Name() == refName {
		return "", fmt.Errorf("cannot delete branch '%s' checked out at current worktree", name)
	}

	// Check merged status if not force (skip for simulation simplicity mostly, unless requested)
	// For now, allow delete if force or if we implement merge check.
	// Assuming force=true implies skip check.

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
	return `usage: git branch [options] [-r | -a] [--merged | --no-merged]
       git branch [options] [-l] [-f] <branchname> [<start-point>]
       git branch [options] [-r] (-d | -D) <branchname>...
       git branch [options] (-m | -M) [<oldbranch>] <newbranch>

Options:
    -d, --delete          delete fully merged branch
    -D                    delete branch (even if not merged)
    -m, --move            move/rename a branch and its reflog
    -M                    move/rename a branch, even if target exists
    -c, --copy            copy a branch and its reflog
    -C                    copy a branch, even if target exists
    -l, --list            list branch names
    -r, --remotes         act on remote-tracking branches
    -a, --all             list both remote-tracking and local branches
    --help                display this help message
`
}
