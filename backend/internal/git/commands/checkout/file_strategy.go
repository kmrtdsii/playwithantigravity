package checkout

import (
	"fmt"
	"os"

	"github.com/kurobon/gitgym/backend/internal/git"
)

// FileStrategy handles "git checkout -- <file>" operations.
type FileStrategy struct{}

var _ Strategy = (*FileStrategy)(nil)

// Execute restores files from HEAD to the working tree.
func (s *FileStrategy) Execute(sess *git.Session, ctx *Context, _ *Options) (string, error) {
	headRef, err := ctx.Repo.Head()
	if err != nil {
		return "", fmt.Errorf("fatal: cannot checkout file without HEAD")
	}
	headCommit, err := ctx.Repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	for _, filename := range ctx.Files {
		file, err := headCommit.File(filename)
		if err != nil {
			return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", filename)
		}
		content, _ := file.Contents()

		f, err := ctx.Worktree.Filesystem.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return "", err
		}
		_, _ = f.Write([]byte(content))
		_ = f.Close()
	}

	if len(ctx.Files) == 1 {
		return "Updated " + ctx.Files[0], nil
	}
	return fmt.Sprintf("Updated %d files", len(ctx.Files)), nil
}
