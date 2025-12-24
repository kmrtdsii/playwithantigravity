package commands

import (
	"context"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("checkout", func() git.Command { return &CheckoutCommand{} })
}

type CheckoutCommand struct{}

func (c *CheckoutCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := repo.Worktree()

	// Flags
	var (
		newBranch      string
		forceNewBranch string
		orphanBranch   string
		force          bool
		detach         bool // Not explicitly flagged often, but logic might support
		target         string
	)

	// Parse flags
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-b":
			if i+1 >= len(cmdArgs) {
				return "", fmt.Errorf("fatal: missing branch name for -b")
			}
			newBranch = cmdArgs[i+1]
			i++
		case "-B":
			if i+1 >= len(cmdArgs) {
				return "", fmt.Errorf("fatal: missing branch name for -B")
			}
			forceNewBranch = cmdArgs[i+1]
			i++
		case "--orphan":
			if i+1 >= len(cmdArgs) {
				return "", fmt.Errorf("fatal: missing branch name for --orphan")
			}
			orphanBranch = cmdArgs[i+1]
			i++
		case "-f", "--force":
			force = true
		case "--detach":
			detach = true // Explicit detach
		case "-h", "--help":
			return c.Help(), nil
		case "--":
			// End of flags, remainder are paths?
			// git checkout -- <file>
			if i+1 >= len(cmdArgs) {
				return "", fmt.Errorf("fatal: filename required after --")
			}
			return c.checkoutFiles(repo, w, cmdArgs[i+1:])
		default:
			if target == "" {
				target = arg
			}
			// else: multiple args not supported for branch checkout unless paths
		}
	}

	// Logic Dispatch

	if orphanBranch != "" {
		return c.checkoutOrphan(repo, s, orphanBranch)
	}

	if newBranch != "" || forceNewBranch != "" {
		name := newBranch
		forceCreate := false
		if forceNewBranch != "" {
			name = forceNewBranch
			forceCreate = true
		}

		startPoint := target
		if startPoint == "" {
			startPoint = "HEAD"
		}

		return c.createAndCheckout(repo, w, s, name, startPoint, forceCreate, force)
	}

	if target == "" {
		return "", fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch>")
	}

	// Try checking out target as reference/commit
	return c.checkoutRefOrPath(repo, w, s, target, force, detach)
}

func (c *CheckoutCommand) checkoutOrphan(repo *gogit.Repository, s *git.Session, branchName string) (string, error) {
	// orphan branch = unborn branch.
	// We point HEAD to refs/heads/<branchName> but do NOT create the ref.
	// We also strictly preserve index and working tree (which go-git does by default if we don't call Checkout).

	refName := plumbing.ReferenceName("refs/heads/" + branchName)

	// Verify it doesn't exist
	_, err := repo.Reference(refName, true)
	if err == nil {
		return "", fmt.Errorf("fatal: A branch named '%s' already exists.", branchName)
	}

	// Set HEAD to symbolic ref (unborn)
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
	if err := repo.Storer.SetReference(headRef); err != nil {
		return "", fmt.Errorf("failed to set HEAD for orphan: %w", err)
	}

	s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s (orphan)", "HEAD", branchName))

	return fmt.Sprintf("Switched to a new branch '%s' (orphan)", branchName), nil
}

func (c *CheckoutCommand) checkoutFiles(repo *gogit.Repository, w *gogit.Worktree, files []string) (string, error) {
	// Restore files from HEAD
	// Simplified: only support restoring from HEAD
	// "git checkout -- file"

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("fatal: cannot checkout file without HEAD")
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	for _, filename := range files {
		file, err := headCommit.File(filename)
		if err != nil {
			return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", filename)
		}
		content, _ := file.Contents()

		f, _ := w.Filesystem.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		f.Write([]byte(content))
		f.Close()
	}

	if len(files) == 1 {
		return "Updated " + files[0], nil
	}
	return fmt.Sprintf("Updated %d files", len(files)), nil
}

func (c *CheckoutCommand) createAndCheckout(repo *gogit.Repository, w *gogit.Worktree, s *git.Session, branchName, startPoint string, forceCreate, forceCheckout bool) (string, error) {
	// Resolve start point
	hash, err := repo.ResolveRevision(plumbing.Revision(startPoint))
	if err != nil {
		return "", fmt.Errorf("fatal: invalid reference: %s", startPoint)
	}

	// Create branch ref
	refName := plumbing.ReferenceName("refs/heads/" + branchName)

	// Check existence
	_, err = repo.Reference(refName, true)
	if err == nil && !forceCreate {
		return "", fmt.Errorf("fatal: A branch named '%s' already exists.", branchName)
	}

	newRef := plumbing.NewHashReference(refName, *hash)
	if errRef := repo.Storer.SetReference(newRef); errRef != nil {
		return "", errRef
	}

	// Checkout
	opts := &gogit.CheckoutOptions{
		Branch: refName,
		Force:  forceCheckout,
	}
	if errCheckout := w.Checkout(opts); errCheckout != nil {
		return "", errCheckout
	}

	s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", startPoint, branchName))

	if forceCreate && err == nil { // err from check existence was nil aka existed
		return fmt.Sprintf("Reset branch '%s'", branchName), nil
	}
	return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
}

func (c *CheckoutCommand) checkoutRefOrPath(repo *gogit.Repository, w *gogit.Worktree, s *git.Session, target string, force, detach bool) (string, error) {
	// 1. Try as branch (unless --detach)
	if !detach {
		branchRef := plumbing.ReferenceName("refs/heads/" + target)
		err := w.Checkout(&gogit.CheckoutOptions{
			Branch: branchRef,
			Force:  force,
		})
		if err == nil {
			s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
			return fmt.Sprintf("Switched to branch '%s'", target), nil
		}
	}

	// 1.5. Check if it's a remote branch (Auto-track)
	remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", target))
	if remoteRef, err := repo.Reference(remoteRefName, true); err == nil && !detach {
		// Found matching remote branch!
		// Create local branch 'target' pointing to same hash
		newBranchRef := plumbing.ReferenceName("refs/heads/" + target)
		newRef := plumbing.NewHashReference(newBranchRef, remoteRef.Hash())

		if err := repo.Storer.SetReference(newRef); err != nil {
			return "", fmt.Errorf("failed to create tracking branch: %w", err)
		}

		// Checkout the new local branch
		err := w.Checkout(&gogit.CheckoutOptions{
			Branch: newBranchRef,
			Force:  force,
		})
		if err == nil {
			s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
			return fmt.Sprintf("Switched to a new branch '%s'\nBranch '%s' set up to track remote branch '%s' from 'origin'.", target, target, target), nil
		}
	}

	// 2. Try as hash/tag (Detached HEAD)
	hash, err := repo.ResolveRevision(plumbing.Revision(target))
	if err == nil {
		// Verify it's a commit
		if _, errObj := repo.CommitObject(*hash); errObj != nil {
			return "", fmt.Errorf("reference is not a commit: %v", errObj)
		}

		err = w.Checkout(&gogit.CheckoutOptions{
			Hash:  *hash,
			Force: force,
		})
		if err == nil {
			s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
			return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", target), nil
		}
		return "", err
	}

	// 3. Fallback: treat as file path?
	// git checkout <file> shorthand for git checkout -- <file>
	// Check if file exists in HEAD
	// Re-use checkoutFiles logic
	return c.checkoutFiles(repo, w, []string{target})
}

func (c *CheckoutCommand) Help() string {
	return `usage: git checkout [-b|-B <new_branch>] [<start_point>]
       git checkout <branch>
       git checkout <commit>
       git checkout [--] <file>...

Options:
    -b <branch>       create and checkout a new branch
    -B <branch>       create/reset and checkout a branch
    -f, --force       force checkout (throw away local changes)
    --detach          detach HEAD at named commit
    --help            display this help message
`
}
