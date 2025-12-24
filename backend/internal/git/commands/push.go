package commands

// push.go - Simulated Git Push Command
//
// IMPORTANT: This implementation does NOT perform actual network operations.
// It simulates push by copying objects between in-memory repositories:
//   - Session-local repos (s.Repos)
//   - Shared virtual remotes (s.Manager.SharedRemotes)
//
// This is safe for educational/sandbox use. No real GitHub/GitLab push occurs.

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
	isHelp := false
	var positionalArgs []string
	for i, arg := range args {
		if i == 0 {
			continue // skip "push"
		}
		switch arg {
		case "-f", "--force":
			isForce = true
		case "-n", "--dry-run":
			isDryRun = true
		case "-h", "--help":
			isHelp = true
		default:
			if strings.HasPrefix(arg, "-") {
				// ignore other flags for now
			} else {
				positionalArgs = append(positionalArgs, arg)
			}
		}
	}

	if isHelp {
		return c.Help(), nil
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

		// Fallback: Check using full URL (in case lookupKey stripped leading slash but map has it)
		if !ok {
			targetRepo, ok = s.Manager.SharedRemotes[url]
		}
	}

	if !ok {
		// FALLBACK: Check if it is a local filesystem path (persistent remote)
		// We trust the URL from the config.
		var errOpen error
		targetRepo, errOpen = gogit.PlainOpen(url)
		if errOpen == nil {
			ok = true
		} else {
			// Try without leading slash if it was stripped?
			targetRepo, errOpen = gogit.PlainOpen(lookupKey)
			if errOpen == nil {
				ok = true
			}
		}
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
			isFF, err := git.IsFastForward(repo, targetRef.Hash(), refToPush.Hash())
			if err != nil {
				return "", err
			}
			if !isFF {
				return "", fmt.Errorf("non-fast-forward update rejected (use --force to override)")
			}
		}
	} else if refToPush.Name().IsTag() {
		_, err := targetRepo.Reference(refToPush.Name(), true)
		if err == nil && !isForce {
			return "", fmt.Errorf("tag '%s' already exists (use --force to override)", refToPush.Name().Short())
		}
	}

	if isDryRun {
		return fmt.Sprintf("[dry-run] Would push %s to %s at %s", refToPush.Name().Short(), remoteName, url), nil
	}

	// SIMULATE PUSH: Copy Objects + Update Ref
	hashToSync := refToPush.Hash()

	// Check object type
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hashToSync)
	if err != nil {
		return "", err
	}

	if obj.Type() == plumbing.TagObject {
		// It's an annotated tag.
		if !git.HasObject(targetRepo, hashToSync) {
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
		if err := git.CopyCommitRecursive(repo, targetRepo, tagObj.Target); err != nil {
			return "", err
		}
	} else if obj.Type() == plumbing.CommitObject {
		if err := git.CopyCommitRecursive(repo, targetRepo, hashToSync); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("unsupported object type to push: %s", obj.Type())
	}

	// Update Remote Reference
	err = targetRepo.Storer.SetReference(refToPush)
	if err != nil {
		return "", err
	}

	// Update Local Remote-Tracking Reference (ONLY for branches)
	if refToPush.Name().IsBranch() {
		localRemoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, refToPush.Name().Short()))
		newLocalRemoteRef := plumbing.NewHashReference(localRemoteRefName, refToPush.Hash())
		_ = repo.Storer.SetReference(newLocalRemoteRef)
	}

	return fmt.Sprintf("To %s\n   %s -> %s", url, hashToSync.String()[:7], refToPush.Name().Short()), nil
}

func (c *PushCommand) Help() string {
	return `usage: git push [options] [<remote>] [<refspec>]

Options:
    -f, --force       force updates (overwrites non-fast-forward)
    -n, --dry-run     dry run (show what would be pushed without doing it)
    --help            display this help message

Note: This is a simulated push. Objects are copied to in-memory
virtual remotes only. No actual network operations are performed.
`
}
