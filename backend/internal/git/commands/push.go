package commands

import (
	"context"
	"fmt"
	"strings"

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

	// Parse Flags
	isForce := false
	isDryRun := false
	var positionalArgs []string
	for i, arg := range args {
		if i == 0 {
			continue // skip "push"
		}
		if arg == "-f" || arg == "--force" {
			isForce = true
		} else if arg == "-n" || arg == "--dry-run" {
			isDryRun = true
		} else if strings.HasPrefix(arg, "-") {
			// ignore other flags for now
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Syntax: git push [remote] [branch]
	remoteName := "origin"
	if len(positionalArgs) > 0 {
		remoteName = positionalArgs[0]
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

	// Resolve local simulated remote path
	lookupKey := strings.TrimPrefix(url, "/")

	var targetRepo *gogit.Repository
	var ok bool

	// Check Session-local Repos
	targetRepo, ok = s.Repos[lookupKey]
	if !ok && s.Manager != nil {
		// Check Shared Remotes
		targetRepo, ok = s.Manager.SharedRemotes[lookupKey]
	}

	if !ok {
		return "", fmt.Errorf("remote repository '%s' not found (only local simulation supported)", url)
	}

	// Determined Branch to Push
	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not on a branch (detached?)")
	}
	branchName := headRef.Name()

	// Check Fast-Forward (unless Force)
	targetRef, err := targetRepo.Reference(branchName, true)
	if err == nil && !isForce {
		isFF, err := isFastForward(repo, targetRef.Hash(), headRef.Hash())
		if err != nil {
			return "", err
		}
		if !isFF {
			return "", fmt.Errorf("non-fast-forward update rejected (use --force to override)")
		}
	}

	if isDryRun {
		return fmt.Sprintf("[dry-run] Would push %s to %s at %s", branchName.Short(), remoteName, url), nil
	}

	// SIMULATE PUSH: Copy Objects + Update Ref
	err = copyCommitRecursive(repo, targetRepo, headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to push objects: %w", err)
	}

	err = targetRepo.Storer.SetReference(headRef)
	if err != nil {
		return "", err
	}

	// 2. Update Local Remote-Tracking Reference: refs/remotes/<remote>/<branch>
	localRemoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName.Short()))
	newLocalRemoteRef := plumbing.NewHashReference(localRemoteRefName, headRef.Hash())
	err = repo.Storer.SetReference(newLocalRemoteRef)
	if err != nil {
		return "", fmt.Errorf("failed to update remote-tracking reference: %w", err)
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
