package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
)

func init() {
	git.RegisterCommand("push", func() git.Command { return &PushCommand{} })
}

type PushCommand struct{}

func (c *PushCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Syntax: git push [remote] [branch]
	// Defaults: origin, current branch
	remoteName := "origin"
	if len(args) > 1 {
		remoteName = args[1]
	}

	// Resolve Remote URL
	rem, err := repo.Remote(remoteName)
	if err != nil {
		return "", fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
	}

	cfg := rem.Config()
	if len(cfg.URLs) == 0 {
		return "", fmt.Errorf("remote %s has no URL defined", remoteName)
	}
	url := cfg.URLs[0]

	// Check if this is a local simulated remote (in Session.Repos)
	// We matched cleanPath in init.go, so we should look for matches.
	// URLs in config might be relative or absolute.
	// Simplify: Check if `url` exists in s.Repos map directly or relative to CurrentDir keys.
	// But `url` usually is just the path.

	targetRepo, ok := s.Repos[url]
	if !ok {
		// Try resolving relative to current dir?
		// If current dir is /, and url is remote.git, key is remote.git.
		// If current dir is /work, key is work.
		// If init --bare remote.git was called at root, key is "remote.git".
		// So strict match is fine for now.
		return "", fmt.Errorf("remote repository '%s' not found in simulation session (only local simulation supported)", url)
	}

	// Determined Branch to Push
	// For simplicity, push current HEAD to same branch name on remote
	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not on a branch (detached?)")
	}
	branchName := headRef.Name()

	// SIMULATE PUSH: Copy Objects + Update Ref

	// 1. Copy Objects (Naive: Copy ALL? or Walk?)
	// Walking is better.
	// We need to copy commit object and everything reachable that doesn't exist in target.

	// Local Commit
	// (We use headRef.Hash() directly now for recursion, commit object fetch is done inside if needed)

	// 1. Copy Objects (Recursive)
	// We pass *gogit.Repository, not *git.Repository (wrapper)
	// But s.Repos stores *gogit.Repository ?
	// s.Repos map[string]*git.Repository ? No, gogit.Repository?
	// Check session.go or init.go
	// init.go: s.Repos[cleanPath] = repo (repo is *gogit.Repository)
	// state.go: repo := session.GetRepo() -> *gogit.Repository

	// Better strategy: Use ObjectWalker if available? No.
	// Manual recursion:
	// Note: CommitWalker ForEach just iterates. We need custom walker or manual recursion?
	// go-git CommitWalker iterates commits.
	// We need to traverse Trees/Blobs too.

	// Better strategy: Use ObjectWalker if available? No.
	// Manual recursion:
	// 1. Copy Objects (Recursive)
	// We pass *gogit.Repository, not *git.Repository (wrapper)
	// But s.Repos stores *gogit.Repository ?
	// s.Repos map[string]*git.Repository ? No, gogit.Repository?
	// Check session.go or init.go
	// init.go: s.Repos[cleanPath] = repo (repo is *gogit.Repository)
	// state.go: repo := session.GetRepo() -> *gogit.Repository

	err = copyCommitRecursive(repo, targetRepo, headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to push objects: %w", err)
	}

	// 2. Update Remote Ref
	// git push updates refs/heads/<branch> on remote
	// Also need to handle fast-forward check?
	// Simulation: Force update for now or simple check.
	// Standard: verify target ref is ancestor of new ref.

	targetRef, err := targetRepo.Reference(branchName, true)
	if err == nil {
		// Ref exists, check fast-forward
		// If we can reach old hash from new hash, it is FF.
		isFF, err := isFastForward(repo, targetRef.Hash(), headRef.Hash())
		if err != nil {
			return "", err
		}
		if !isFF {
			return "", fmt.Errorf("non-fast-forward update rejected (simulation)")
		}
	}

	err = targetRepo.Storer.SetReference(headRef)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("To %s\n   %s -> %s", url, headRef.Hash().String()[:7], branchName.Short()), nil
}

func (c *PushCommand) Help() string {
	return "usage: git push [remote] [branch]"
}

// Helpers

func copyCommitRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	// Check if dst has it
	if hasObject(dst, hash) {
		return nil
	}

	// Get from Source
	obj, err := src.Storer.EncodedObject(plumbing.CommitObject, hash)
	if err != nil {
		return err
	}

	// Write to Dest
	_, err = dst.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	// Decode to parse tree/parents
	commit, err := object.DecodeCommit(src.Storer, obj)
	if err != nil {
		return err
	}

	// Recurse parents
	for _, p := range commit.ParentHashes {
		if err := copyCommitRecursive(src, dst, p); err != nil {
			return err
		}
	}

	// Copy Tree
	return copyTreeRecursive(src, dst, commit.TreeHash)
}

func copyTreeRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if hasObject(dst, hash) {
		return nil
	}

	obj, err := src.Storer.EncodedObject(plumbing.TreeObject, hash)
	if err != nil {
		return err
	}

	_, err = dst.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	tree, err := object.DecodeTree(src.Storer, obj)
	if err != nil {
		return err
	}

	for _, entry := range tree.Entries {
		if entry.Mode == 0160000 {
			// Submodule (commit), ignore or handle?
			continue
		}
		if entry.Mode.IsFile() {
			if err := copyBlob(src, dst, entry.Hash); err != nil {
				return err
			}
		} else {
			if err := copyTreeRecursive(src, dst, entry.Hash); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyBlob(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if hasObject(dst, hash) {
		return nil
	}
	obj, err := src.Storer.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		return err
	}
	_, err = dst.Storer.SetEncodedObject(obj)
	return err
}

func hasObject(repo *gogit.Repository, hash plumbing.Hash) bool {
	// HasEncodedObject might be faster check
	_, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	return err == nil
}

// Unused legacy helpers removed
// func copyObject(src, dst *gogit.Repository, hash plumbing.Hash) error { return nil }
// func copyTree(src, dst *gogit.Repository, hash plumbing.Hash) error { return nil }

// isFastForward moved to utils.go
