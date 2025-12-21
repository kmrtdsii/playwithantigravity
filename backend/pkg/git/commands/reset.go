package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("reset", func() git.Command { return &ResetCommand{} })
}

type ResetCommand struct{}

func (c *ResetCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// git reset [<mode>] [<commit>]
	// modes: --soft, --mixed, --hard
	// default mixed
	mode := gogit.MixedReset
	target := "HEAD"

	argsIdx := 1
	if len(args) > argsIdx && strings.HasPrefix(args[argsIdx], "--") {
		switch args[argsIdx] {
		case "--soft":
			mode = gogit.SoftReset
		case "--mixed":
			mode = gogit.MixedReset
		case "--hard":
			mode = gogit.HardReset
		default:
			return "", fmt.Errorf("unknown reset mode: %s", args[argsIdx])
		}
		argsIdx++
	}

	if len(args) > argsIdx {
		target = args[argsIdx]
	}

	// Resolve target
	h, err := repo.ResolveRevision(plumbing.Revision(target))
	if err != nil {
		return "", err
	}

	w, _ := repo.Worktree()
	
	// Update ORIG_HEAD before reset
	s.UpdateOrigHead()

	if err := w.Reset(&gogit.ResetOptions{
		Commit: *h,
		Mode:   mode,
	}); err != nil {
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("reset: moving to %s", target))

	return fmt.Sprintf("HEAD is now at %s", h.String()[:7]), nil
}

func (c *ResetCommand) Help() string {
	return "usage: git reset [--soft|mixed|hard] <commit>"
}
