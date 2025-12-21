package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kmrtdsii/playwithantigravity/backend/pkg/git"
)

func init() {
	git.RegisterCommand("clone", func() git.Command { return &CloneCommand{} })
}

type CloneCommand struct{}

func (c *CloneCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git clone <url>")
	}
	url := args[1]

	if s.Repo != nil {
		return "", fmt.Errorf("repository already initialized")
	}

	st := memory.NewStorage()
	repo, err := gogit.Clone(st, s.Filesystem, &gogit.CloneOptions{
		URL:      url,
		Progress: nil, // TODO: Maybe wire up progress to stdout?
	})
	if err != nil {
		return "", fmt.Errorf("clone failed: %w", err)
	}

	s.Repo = repo

	// We might want to set ORIG_HEAD or similar, but Clone usually sets everything up.

	return fmt.Sprintf("Cloned into '.' from %s", url), nil
}

func (c *CloneCommand) Help() string {
	return "usage: git clone <url>\n\nClone a repository into a new directory." // We actually clone into root of filesystem for this session
}
