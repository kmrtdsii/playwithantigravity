package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
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

	// Syntax: git push [remote] [refspec]
	remoteName := "origin"
	refspec := ""
	if len(positionalArgs) > 0 {
		remoteName = positionalArgs[0]
	}
	if len(positionalArgs) > 1 {
		refspec = positionalArgs[1]
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

	// Determined Ref to Push
	var refToPush *plumbing.Reference

	if refspec != "" {
		// Try to resolve refspec (Branch or Tag)
		// 1. Try exact match
		ref, err := repo.Reference(plumbing.ReferenceName(refspec), true)
		if err == nil {
			refToPush = ref
		} else {
			// 2. Try refs/heads/
			ref, err = repo.Reference(plumbing.ReferenceName("refs/heads/"+refspec), true)
			if err == nil {
				refToPush = ref
			} else {
				// 3. Try refs/tags/
				ref, err = repo.Reference(plumbing.ReferenceName("refs/tags/"+refspec), true)
				if err == nil {
					refToPush = ref
				} else {
					return "", fmt.Errorf("src refspec '%s' does not match any", refspec)
				}
			}
		}
	} else {
		// Default: Push HEAD
		headRef, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
		if !headRef.Name().IsBranch() {
			return "", fmt.Errorf("HEAD is not on a branch (detached?)")
		}
		refToPush = headRef
	}

	// Check Fast-Forward (only for branches)
	if refToPush.Name().IsBranch() && !isForce {
		targetRef, err := targetRepo.Reference(refToPush.Name(), true)
		if err == nil {
			isFF, err := isFastForward(repo, targetRef.Hash(), refToPush.Hash())
			if err != nil {
				return "", err
			}
			if !isFF {
				return "", fmt.Errorf("non-fast-forward update rejected (use --force to override)")
			}
		}
	} else if refToPush.Name().IsTag() {
		// For tags, check if it exists and differs (conflict) unless force?
		// Git usually rejects existing tag overwrite unless force.
		_, err := targetRepo.Reference(refToPush.Name(), true)
		if err == nil && !isForce {
			return "", fmt.Errorf("tag '%s' already exists (use --force to override)", refToPush.Name().Short())
		}
	}

	if isDryRun {
		return fmt.Sprintf("[dry-run] Would push %s to %s at %s", refToPush.Name().Short(), remoteName, url), nil
	}

	// SIMULATE PUSH: Copy Objects + Update Ref
	// If it's a tag, we might need to copy the Tag Object itself + the Commit it points to.
	// copyCommitRecursive calls copyTreeRecursive.
	// If it's an annotated tag, the Ref points to a Tag Object -> Commit.

	// We need generic Object Copy logic that follows dependencies.
	// Current copyCommitRecursive starts at a Commit Hash.

	hashToSync := refToPush.Hash()

	// Check object type
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hashToSync)
	if err != nil {
		return "", err
	}

	if obj.Type() == plumbing.TagObject {
		// It's an annotated tag.
		// Copy tag object
		if !hasObject(targetRepo, hashToSync) {
			_, err = targetRepo.Storer.SetEncodedObject(obj)
			if err != nil {
				return "", err
			}
		}
		// Decode tag to find target commit
		tagObj, err := object.DecodeTag(repo.Storer, obj)
		if err != nil {
			return "", err
		}

		// Recursively copy the commit it points to
		if err := copyCommitRecursive(repo, targetRepo, tagObj.Target); err != nil {
			return "", err
		}
	} else if obj.Type() == plumbing.CommitObject {
		if err := copyCommitRecursive(repo, targetRepo, hashToSync); err != nil {
			return "", err
		}
	} else {
		// Blob or Tree? Unlikely for a ref push but possible.
		return "", fmt.Errorf("unsupported object type to push: %s", obj.Type())
	}

	// Update Remote Reference
	err = targetRepo.Storer.SetReference(refToPush)
	if err != nil {
		return "", err
	}

	// Update Local Remote-Tracking Reference (ONLY for branches)
	// e.g. refs/remotes/origin/main
	if refToPush.Name().IsBranch() {
		localRemoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, refToPush.Name().Short()))
		newLocalRemoteRef := plumbing.NewHashReference(localRemoteRefName, refToPush.Hash())
		_ = repo.Storer.SetReference(newLocalRemoteRef) // Ignore error if fails?
	}
	// For tags, we don't usually create "remote-tracking tags" in refs/remotes. Tags are shared.

	return fmt.Sprintf("To %s\n   %s -> %s", url, hashToSync.String()[:7], refToPush.Name().Short()), nil
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
