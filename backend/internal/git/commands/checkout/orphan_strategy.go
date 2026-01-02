package checkout

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

// OrphanStrategy handles "git checkout --orphan <branch>" operations.
type OrphanStrategy struct{}

var _ Strategy = (*OrphanStrategy)(nil)

// Execute creates an orphan branch (a branch with no parent commits).
func (s *OrphanStrategy) Execute(sess *git.Session, ctx *Context, _ *Options) (string, error) {
	refName := plumbing.ReferenceName("refs/heads/" + ctx.OrphanBranch)
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
	if err := ctx.Repo.Storer.SetReference(headRef); err != nil {
		return "", fmt.Errorf("failed to set HEAD for orphan: %w", err)
	}

	sess.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s (orphan)", "HEAD", ctx.OrphanBranch))
	return fmt.Sprintf("Switched to a new branch '%s' (orphan)", ctx.OrphanBranch), nil
}
