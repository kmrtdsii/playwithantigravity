package checkout

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// RefStrategy handles "git checkout <ref>" operations (branch, tag, commit).
type RefStrategy struct{}

var _ Strategy = (*RefStrategy)(nil)

// Execute switches to an existing branch, tag, or commit.
func (s *RefStrategy) Execute(sess *git.Session, ctx *Context, opts *Options) (string, error) {
	gOpts := &gogit.CheckoutOptions{Force: opts.Force}

	if ctx.TargetRef != "" {
		if ctx.TargetRef.IsRemote() {
			// Create local branch tracking remote
			localName := opts.Target
			localRef := plumbing.ReferenceName("refs/heads/" + localName)
			newRef := plumbing.NewHashReference(localRef, *ctx.TargetHash)
			if err := ctx.Repo.Storer.SetReference(newRef); err != nil {
				return "", err
			}
			gOpts.Branch = localRef
		} else {
			gOpts.Branch = ctx.TargetRef
		}
	} else if ctx.TargetHash != nil {
		gOpts.Hash = *ctx.TargetHash
	}

	if err := ctx.Worktree.Checkout(gOpts); err != nil {
		return "", err
	}

	sess.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", opts.Target))

	if ctx.IsDetached {
		return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", opts.Target), nil
	}
	if ctx.TargetRef != "" && ctx.TargetRef.IsRemote() {
		return fmt.Sprintf("Switched to a new branch '%s'\nBranch '%s' set up to track remote branch '%s' from 'origin'.", opts.Target, opts.Target, opts.Target), nil
	}
	return fmt.Sprintf("Switched to branch '%s'", opts.Target), nil
}
