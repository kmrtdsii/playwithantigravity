package commands

// commit.go - Simulated Git Commit Command
//
// Records changes to the repository by creating a new commit object.
// Supports -m (message), --amend, and --allow-empty flags.

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("commit", func() git.Command { return &CommitCommand{} })
}

type CommitCommand struct{}

// Ensure CommitCommand implements git.Command
var _ git.Command = (*CommitCommand)(nil)

type CommitOptions struct {
	Message    string
	Amend      bool
	AllowEmpty bool
}

type commitContext struct {
	w           *gogit.Worktree
	repo        *gogit.Repository
	message     string
	amendCommit *object.Commit
}

func (c *CommitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Parse
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	// 2. Resolve
	cCtx, err := c.resolveContext(repo, opts, args)
	if err != nil {
		return "", err
	}

	// 3. Perform
	return c.performAction(s, cCtx, opts)
}

func (c *CommitCommand) parseArgs(args []string) (*CommitOptions, error) {
	opts := &CommitOptions{
		// Message: "", // Default is empty
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		case "-m":
			if i+1 < len(args) {
				opts.Message = args[i+1]
				i++
			}
		case "--amend":
			opts.Amend = true
		case "--allow-empty":
			opts.AllowEmpty = true
		case "--no-edit":
			// Shim: In GitGym, amending without -m automatically behaves like --no-edit
			// We just accept the flag to avoid error.
		default:
			// Reject positional arguments or unknown flags
			// Standard git treats positional args as file paths, but we don't fully support that yet.
			// Even if we did, "git commit --amend <text>" is usually an error (text interpreted as path).
			return nil, fmt.Errorf("unknown argument or option: '%s'. Did you mean to use -m for message?", arg)
		}
	}
	return opts, nil
}

func (c *CommitCommand) resolveContext(repo *gogit.Repository, opts *CommitOptions, originalArgs []string) (*commitContext, error) {
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	ctx := &commitContext{
		w:    w,
		repo: repo,
	}

	if opts.Amend {
		headRef, err := repo.Head()
		if err != nil {
			return nil, fmt.Errorf("cannot amend without HEAD: %v", err)
		}
		headCommit, err := repo.CommitObject(headRef.Hash())
		if err != nil {
			return nil, err
		}
		ctx.amendCommit = headCommit

		// Handle message reuse for amend
		isMsgProvided := false
		for _, arg := range originalArgs {
			if arg == "-m" {
				isMsgProvided = true
				break
			}
		}

		if isMsgProvided {
			ctx.message = opts.Message
		} else {
			ctx.message = headCommit.Message
		}
	} else {
		// Normal Commit: Message is REQUIRED
		if opts.Message == "" {
			return nil, fmt.Errorf("message is required. Use -m \"message\"")
		}
		ctx.message = opts.Message
	}

	return ctx, nil
}

func (c *CommitCommand) performAction(s *git.Session, ctx *commitContext, opts *CommitOptions) (string, error) {
	var commitOpts gogit.CommitOptions
	commitOpts.Author = git.GetDefaultSignature()
	commitOpts.AllowEmptyCommits = opts.AllowEmpty

	actionLabel := "commit"

	if opts.Amend {
		s.UpdateOrigHead()
		commitOpts.Parents = ctx.amendCommit.ParentHashes
		commitOpts.AllowEmptyCommits = true // Amending generally allowed
		actionLabel = "commit (amend)"
	}

	commitHash, err := ctx.w.Commit(ctx.message, &commitOpts)
	if err != nil {
		if strings.Contains(err.Error(), "clean") || strings.Contains(err.Error(), "nothing to commit") {
			return "", fmt.Errorf("%v\nhint: Use 'git commit --allow-empty -m <message>' to create an empty commit", err)
		}
		return "", err
	}

	s.RecordReflog(fmt.Sprintf("%s: %s", actionLabel, strings.Split(ctx.message, "\n")[0]))

	if opts.Amend {
		return fmt.Sprintf("Commit amended: %s", commitHash.String()), nil
	}
	return fmt.Sprintf("Commit created: %s", commitHash.String()), nil
}

func (c *CommitCommand) Help() string {
	return `📘 GIT-COMMIT (1)                                       Git Manual

 💡 DESCRIPTION
    ・ステージングエリアにある変更を記録する（セーブする）
    ・変更内容にメッセージを付けて保存する

 📋 SYNOPSIS
    git commit -m <msg> [--amend] [--allow-empty]

 ⚙️  COMMON OPTIONS
    -m <msg>
        コミットメッセージを指定します。

    --amend
        直前のコミットを修正します（メッセージの変更や、ファイルの追加忘れ等）。
        ※ Push済みのコミットに対して行うと履歴が壊れるため、Push前だけに行いましょう。

    --allow-empty
        変更が含まれていなくてもコミットを作成できるようにします。

 🛠  PRACTICAL EXAMPLES
    1. 基本: メッセージ付きでコミット
       1コミットにつき1つの論点（変更理由）になるよう意識するのがコツです。
       $ git commit -m "feat: add user endpoint"

    2. 実践: 直前のコミットを修正 (Recommended)
       「あっ、メッセージ間違えた！」という時に使います。
       Push前であれば、履歴を汚さずにこっそり直せます。
       $ git commit --amend -m "fix: typo in endpoint"

    3. 実践: ファイルの入れ忘れを修正 + メッセージ変更
       ファイルを追加し忘れた場合も --amend で修正できます。
       $ git add forgotten_file.go
       $ git commit --amend -m "fix: add user endpoint"

       (メッセージはそのままで良い場合)
       $ git commit --amend --no-edit

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-commit
`
}
