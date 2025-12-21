package commands

import (
	"context"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("checkout", func() git.Command { return &CheckoutCommand{} })
	git.RegisterCommand("switch", func() git.Command { return &SwitchCommand{} })
}

type CheckoutCommand struct{}

func (c *CheckoutCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, _ := s.Repo.Worktree()
	if len(args) < 2 {
		return "", fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch> | git checkout -- <file>")
	}

	// Handle file checkout (git checkout -- <file>)
	if args[1] == "--" {
		if len(args) < 3 {
			return "", fmt.Errorf("filename required after --")
		}
		filename := args[2]

		// Restore file from HEAD
		headRef, err := s.Repo.Head()
		if err == nil {
			headCommit, _ := s.Repo.CommitObject(headRef.Hash())
			file, err := headCommit.File(filename)
			if err != nil {
				return "", fmt.Errorf("file %s not found in HEAD", filename)
			}
			content, _ := file.Contents()
			
			f, _ := s.Filesystem.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			f.Write([]byte(content))
			f.Close()
			return "Updated " + filename, nil
		}
		return "", fmt.Errorf("cannot checkout file without HEAD")
	}

	// Handle -B (Force create/reset branch)
	if args[1] == "-B" {
		if len(args) < 3 {
			return "", fmt.Errorf("usage: git checkout -B <branch>")
		}
		branchName := args[2]

		// Get current HEAD to set the branch to
		headRef, err := s.Repo.Head()
		if err != nil {
			return "", fmt.Errorf("cannot checkout -B without HEAD")
		}
		
		// Create/Update reference
		refName := plumbing.ReferenceName("refs/heads/" + branchName)
		newRef := plumbing.NewHashReference(refName, headRef.Hash())
		
		if err := s.Repo.Storer.SetReference(newRef); err != nil {
			return "", err
		}
		
		// Checkout the branch
		opts := &gogit.CheckoutOptions{
			Create: false, // Already created manually
			Force:  false,
			Branch: refName,
		}
		if err := w.Checkout(opts); err != nil {
			return "", err
		}
		s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
		return fmt.Sprintf("Switched to and reset branch '%s'", branchName), nil
	}

	// Handle -b
	if args[1] == "-b" {
		if len(args) < 3 {
			return "", fmt.Errorf("usage: git checkout -b <branch>")
		}
		branchName := args[2]

		opts := &gogit.CheckoutOptions{
			Create: true,
			Force:  false,
			Branch: plumbing.ReferenceName("refs/heads/" + branchName),
		}
		if err := w.Checkout(opts); err != nil {
			return "", err
		}
		s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
		return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
	}

	// Handle normal checkout (branch or commit)
	target := args[1]

	// 1. Try as branch
	branchRef := plumbing.ReferenceName("refs/heads/" + target)
	err := w.Checkout(&gogit.CheckoutOptions{
		Branch: branchRef,
	})
	if err == nil {
		s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
		return fmt.Sprintf("Switched to branch '%s'", target), nil
	}

	// 2. Try as hash (Detached HEAD) / Tag / Short Hash
	// Use ResolveRevision to handle short hashes, tags, etc. properly AND verify existence.
	hash, err := s.Repo.ResolveRevision(plumbing.Revision(target))
	if err == nil {
		// Verify it's a commit
		if _, err := s.Repo.CommitObject(*hash); err != nil {
			return "", fmt.Errorf("reference is not a commit: %v", err)
		}
		
		err = w.Checkout(&gogit.CheckoutOptions{
			Hash: *hash,
		})
		if err == nil {
			s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
			return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", target), nil
		}
		return "", err
	}

	return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", target)
}

func (c *CheckoutCommand) Help() string {
	return "usage: git checkout [-b] <branch>"
}


type SwitchCommand struct{}

func (c *SwitchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if s.Repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	w, _ := s.Repo.Worktree()
	if len(args) < 2 {
		return "", fmt.Errorf("usage: git switch [-c] <branch>")
	}

	// Handle -c (create and switch)
	if args[1] == "-c" {
		if len(args) < 3 {
			return "", fmt.Errorf("usage: git switch -c <branch>")
		}
		branchName := args[2]

		// Create new branch logic (similar to checkout -b)
		opts := &gogit.CheckoutOptions{
			Create: true,
			Force:  false,
			Branch: plumbing.ReferenceName("refs/heads/" + branchName),
		}
		if err := w.Checkout(opts); err != nil {
			return "", err
		}
		s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
		return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
	}

	// Handle normal switch (existing branch)
	target := args[1]
	
	// Validate that target is actually a branch (local)
	branchRefName := "refs/heads/" + target
	_, err := s.Repo.Reference(plumbing.ReferenceName(branchRefName), true)
	if err != nil {
		return "", fmt.Errorf("invalid reference: %s", target)
	}

	branchRef := plumbing.ReferenceName(branchRefName)
	err = w.Checkout(&gogit.CheckoutOptions{
		Branch: branchRef,
	})
	if err == nil {
		s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
		return fmt.Sprintf("Switched to branch '%s'", target), nil
	}
	return "", err
}

func (c *SwitchCommand) Help() string {
	return "usage: git switch [-c] <branch>"
}
