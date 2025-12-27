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

type TagOptions struct {
	List      bool
	Delete    bool
	Annotated bool
	Message   string
	TagName   string
	Commit    string
}

func (c *TagCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	if opts.Delete {
		return c.deleteTag(repo, opts)
	}
	if opts.TagName != "" {
		return c.createTag(repo, opts)
	}
	return c.listTags(repo)
}

func (c *TagCommand) parseArgs(args []string) (*TagOptions, error) {
	opts := &TagOptions{}
	cmdArgs := args[1:]

	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-d", "--delete":
			opts.Delete = true
		case "-a", "--annotate":
			opts.Annotated = true
		case "-m", "--message":
			if i+1 < len(cmdArgs) {
				opts.Message = cmdArgs[i+1]
				i++
			}
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if opts.TagName == "" {
				opts.TagName = arg
			} else if opts.Commit == "" {
				opts.Commit = arg
			}
		}
	}
	return opts, nil
}

func (c *TagCommand) listTags(repo *gogit.Repository) (string, error) {
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

func (c *TagCommand) deleteTag(repo *gogit.Repository, opts *TagOptions) (string, error) {
	if opts.TagName == "" {
		return "", fmt.Errorf("tag name required")
	}
	if err := repo.DeleteTag(opts.TagName); err != nil {
		return "", err
	}
	return "Deleted tag " + opts.TagName, nil
}

func (c *TagCommand) createTag(repo *gogit.Repository, opts *TagOptions) (string, error) {
	var targetRef *plumbing.Reference
	var err error

	if opts.Commit != "" {
		// Resolve commit
		h, err := repo.ResolveRevision(plumbing.Revision(opts.Commit))
		if err != nil {
			return "", err
		}
		// We need a HashReference for CreateTag? No, CreateTag helper in go-git takes hash.
		// For Lightweight tag, SetReference needs a ref.
		// Let's create a temp ref object for target.
		targetRef = plumbing.NewHashReference("refs/heads/temp", *h) // Dummy ref name just for hash carrier?
	} else {
		targetRef, err = repo.Head()
		if err != nil {
			return "", err
		}
	}

	if opts.Annotated {
		msg := opts.Message
		if msg == "" {
			msg = "Tag message"
		}
		_, err = repo.CreateTag(opts.TagName, targetRef.Hash(), &gogit.CreateTagOptions{
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
		return "Created annotated tag " + opts.TagName, nil
	}

	// Lightweight
	refName := plumbing.ReferenceName("refs/tags/" + opts.TagName)
	ref := plumbing.NewHashReference(refName, targetRef.Hash())
	if err := repo.Storer.SetReference(ref); err != nil {
		return "", err
	}
	return "Created tag " + opts.TagName, nil
}

func (c *TagCommand) Help() string {
	return `📘 GIT-TAG (1)                                          Git Manual

 💡 DESCRIPTION
    タグ（コミットにつける名前・目印）に関する以下の操作を行います：
    ・タグの一覧を表示する（引数なし）
    ・新しいタグを作成する
    ・不要なタグを削除する（-d）

 📋 SYNOPSIS
    git tag [-a] [-m <msg>] <tagname> [<commit>]
    git tag -d <tagname>

 ⚙️  COMMON OPTIONS
    -a
        注釈付き（Annotated）タグを作成します。作成者や日時などの情報を含めます。

    -m <msg>
        タグのメッセージを指定します。

    -d
        タグを削除します。

 🛠  EXAMPLES
    1. 軽量タグを作成（現在のHEADに）
       $ git tag v1.0

    2. 注釈付きタグを作成
       $ git tag -a v1.0 -m "Release version 1.0"
`
}
