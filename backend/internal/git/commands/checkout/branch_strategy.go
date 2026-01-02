package checkout

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// BranchStrategy handles "git checkout -b/-B <branch>" operations.
type BranchStrategy struct{}

var _ Strategy = (*BranchStrategy)(nil)

// Execute creates a new branch and checks it out.
func (s *BranchStrategy) Execute(sess *git.Session, ctx *Context, opts *Options) (string, error) {
	refName := plumbing.ReferenceName("refs/heads/" + ctx.NewBranch)
	newRef := plumbing.NewHashReference(refName, *ctx.StartPointHash)
	if err := ctx.Repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	err := ctx.Worktree.Checkout(&gogit.CheckoutOptions{
		Branch: refName,
		Force:  opts.Force,
	})
	if err != nil {
		return "", err
	}

	sess.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", ctx.NewBranch))
	if ctx.ForceCreate {
		return fmt.Sprintf("Reset branch '%s'", ctx.NewBranch), nil
	}
	return fmt.Sprintf("Switched to a new branch '%s'", ctx.NewBranch), nil
}
