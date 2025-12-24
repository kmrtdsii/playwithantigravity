package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("tag", func() git.Command { return &TagCommand{} })
}

type TagCommand struct{}

func (c *TagCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// List tags
	if len(args) == 1 {
		tags, err := repo.Tags()
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		err = tags.ForEach(func(r *plumbing.Reference) error {
			sb.WriteString(r.Name().Short() + "\n")
			return nil
		})
		if err != nil {
			return "", err
		}
		return sb.String(), nil
	}

	// Delete tag
	if args[1] == "-d" {
		if len(args) < 3 {
			return "", fmt.Errorf("tag name required")
		}
		tagName := args[2]
		if err := repo.DeleteTag(tagName); err != nil {
			return "", err
		}
		return "Deleted tag " + tagName, nil
	}

	// Create Tag
	// Check for options
	if args[1] == "-a" {
		if len(args) < 4 {
			return "", fmt.Errorf("tag name and message required for annotated tag") // usage: git tag -a v1 -m "msg"
		}
		tagName := args[2]
		msg := "Tag message"
		if len(args) >= 5 && args[3] == "-m" {
			msg = args[4]
		}
		headRef, err := repo.Head()
		if err != nil {
			return "", err
		}
		_, err = repo.CreateTag(tagName, headRef.Hash(), &gogit.CreateTagOptions{
			Message: msg,
			Tagger: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return "", err
		}
		return "Created annotated tag " + tagName, nil
	}

	// Lightweight tag
	tagName := args[1]
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	refName := plumbing.ReferenceName("refs/tags/" + tagName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		return "", err
	}
	return "Created tag " + tagName, nil
}

func (c *TagCommand) Help() string {
	return "usage: git tag [-d] <tagname> | -a <tagname> -m <msg>"
}
