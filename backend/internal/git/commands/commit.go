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
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("commit", func() git.Command { return &CommitCommand{} })
}

type CommitCommand struct{}

type CommitOptions struct {
	Message    string
	Amend      bool
	AllowEmpty bool
}

func (c *CommitCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// 1. Parse Args
	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// 2. Execution
	return c.executeCommit(s, repo, w, opts, args)
}

func (c *CommitCommand) parseArgs(args []string) (*CommitOptions, error) {
	opts := &CommitOptions{
		Message: "Default commit message",
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
		}
	}
	return opts, nil
}

func (c *CommitCommand) executeCommit(s *git.Session, repo *gogit.Repository, w *gogit.Worktree, opts *CommitOptions, originalArgs []string) (string, error) {
	if opts.Amend {
		return c.handleAmend(s, repo, w, opts, originalArgs)
	}

	// Normal commit
	commit, err := w.Commit(opts.Message, &gogit.CommitOptions{
		Author:            git.GetDefaultSignature(),
		AllowEmptyCommits: opts.AllowEmpty,
	})
	if err != nil {
		if strings.Contains(err.Error(), "clean") || strings.Contains(err.Error(), "nothing to commit") {
			return "", fmt.Errorf("%v\nhint: Use 'git commit --allow-empty -m <message>' to create an empty commit", err)
		}
		return "", err
	}
	s.RecordReflog(fmt.Sprintf("commit: %s", strings.Split(opts.Message, "\n")[0]))
	return fmt.Sprintf("Commit created: %s", commit.String()), nil
}

func (c *CommitCommand) handleAmend(s *git.Session, repo *gogit.Repository, w *gogit.Worktree, opts *CommitOptions, args []string) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("cannot amend without HEAD: %v", err)
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	parents := headCommit.ParentHashes

	// Reuse message if not provided explicitly
	// We check if "Message" changed from default?
	// Or check if -m was present in args?
	// Naive check: if opts.Message is "Default commit message" AND -m wasn't in args...
	// Better: parseArgs logic assumes default.
	// Let's replicate strict logic: check if -m was present in args.
	isMsgProvided := false
	for _, arg := range args {
		if arg == "-m" {
			isMsgProvided = true
			break
		}
	}

	msg := opts.Message
	if !isMsgProvided {
		msg = headCommit.Message
	}

	s.UpdateOrigHead()

	newCommitHash, err := w.Commit(msg, &gogit.CommitOptions{
		Parents:           parents,
		Author:            git.GetDefaultSignature(),
		AllowEmptyCommits: true, // Amending generally allowed
	})
	if err != nil {
		return "", err
	}
	s.RecordReflog("commit (amend): " + strings.Split(msg, "\n")[0])

	return fmt.Sprintf("Commit amended: %s", newCommitHash.String()), nil
}

func (c *CommitCommand) Help() string {
	return `📘 GIT-COMMIT (1)                                       Git Manual

 💡 DESCRIPTION
    ・ステージングエリアにある変更を記録する（セーブする）
    ・変更内容にメッセージを付けて保存する

 📋 SYNOPSIS
    git commit -m <msg>
    git commit --amend
    git commit --allow-empty

 ⚙️  COMMON OPTIONS
    -m <msg>
        コミットメッセージを指定します。

    --amend
        直前のコミットを修正します（メッセージの変更や、ファイルの追加忘れ等）。
        元のコミットは上書きされます。

    --allow-empty
        変更が含まれていなくてもコミットを作成できるようにします。

 🛠  EXAMPLES
    1. メッセージ付きでコミット
       $ git commit -m "Initial commit"

    2. 直前のコミットメッセージを修正
       $ git commit --amend -m "Corrected message"

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-commit
`
}
