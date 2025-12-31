package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("revert", func() git.Command { return &RevertCommand{} })
}

type RevertCommand struct{}

// Ensure RevertCommand implements git.Command
var _ git.Command = (*RevertCommand)(nil)

func (c *RevertCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	// Parse flags and arguments
	var rev string
	var mainline int

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "-m" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("option -m requires a value")
			}
			n, err := fmt.Sscanf(args[i+1], "%d", &mainline)
			if err != nil || n != 1 {
				return "", fmt.Errorf("invalid mainline parent number: %s", args[i+1])
			}
			i++ // skip value
		} else {
			rev = arg
		}
	}

	if rev == "" {
		return "", fmt.Errorf("usage: git revert [-m parent-number] <commit>")
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Resolve Target Commit
	hash, err := git.ResolveRevision(repo, rev)
	if err != nil {
		return "", fmt.Errorf("invalid revision '%s': %v", rev, err)
	}
	targetCommit, err := repo.CommitObject(*hash)
	if err != nil {
		return "", fmt.Errorf("fatal: could not parse commit %s", hash.String())
	}

	// 2. Identify Parent (Theirs/Target state for the revert)
	var parentCommit *object.Commit

	if targetCommit.NumParents() > 1 {
		if mainline == 0 {
			return "", fmt.Errorf("error: commit %s is a merge but no -m option was given", hash.String()[:7])
		}
		if mainline < 1 || mainline > targetCommit.NumParents() {
			return "", fmt.Errorf("error: commit %s does not have parent %d", hash.String()[:7], mainline)
		}
		// Parents are 0-indexed in API, 1-indexed in CLI
		parentCommit, err = targetCommit.Parent(mainline - 1)
		if err != nil {
			return "", err
		}
	} else {
		if mainline != 0 {
			return "", fmt.Errorf("error: mainline was specified but commit %s is not a merge", hash.String()[:7])
		}
		if targetCommit.NumParents() > 0 {
			parentCommit, err = targetCommit.Parent(0)
			if err != nil {
				return "", err
			}
		}
	}
	// Reverting a root commit?
	// "Theirs" should be an empty tree.
	// For simplicity, let's treat it as nil und handle in Merge3Way if it supports it,
	// or create an empty dummy commit if needed.
	// git.Merge3Way usually expects *object.Commit.
	// If implementation supports nil as "empty tree", good. If not, we block root revert for now.
	// Looking at cherry-pick: logic handles Base=nil.
	// Here Parent is "Theirs".

	// 3. Get HEAD (Ours)
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// 4. Execute 3-Way Merge
	// We want to apply the DIFF from Target -> Parent.
	// Base = Target
	// Theirs = Parent (Target^)
	// Ours = HEAD

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// Case: Root Revert
	// If parentCommit is nil, we are reverting to "nothing" (deletion of everything added in root).
	// Merge3Way(w, Base=Target, Ours=Head, Theirs=nil)
	// We need to verify if Merge3Way handles nil.
	// Assuming `git.Merge3Way` (internal helper) handles it or we skip root revert for MVP.
	if parentCommit == nil {
		return "", fmt.Errorf("reverting a root commit is not yet supported in this simulation")
	}

	err = git.Merge3Way(w, targetCommit, headCommit, parentCommit)
	if err != nil {
		if err == git.ErrConflict {
			return "", fmt.Errorf("error: could not revert %s... %s\nhint: after resolving conflicts, commit result", hash.String()[:7], targetCommit.Message)
		}
		return "", fmt.Errorf("failed to revert: %v", err)
	}

	// 5. Commit
	// Standard git revert message
	msg := fmt.Sprintf("Revert \"%s\"\n\nThis reverts commit %s.", strings.TrimSpace(targetCommit.Message), targetCommit.Hash.String())

	// Resolve Author from config
	authorName := "GitGym User"
	authorEmail := "user@gitgym.com"
	if cfg, err := repo.Config(); err == nil {
		if cfg.User.Name != "" {
			authorName = cfg.User.Name
		}
		if cfg.User.Email != "" {
			authorEmail = cfg.User.Email
		}
	}

	newHash, err := w.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Revert successful. New commit %s", newHash.String()[:7]), nil
}

func (c *RevertCommand) Help() string {
	return `📘 GIT-REVERT (1)                                       Git Manual

 💡 DESCRIPTION
    ・既存のコミットを「打ち消す」新しいコミットを作成します。
    ・履歴を改変せず（resetと異なり）、安全に過去の変更を取り消せます。

 📋 SYNOPSIS
    git revert [-m parent-number] <commit>

 ⚙️  OPTIONS
    -m parent-number
        マージコミットを打ち消す場合に、どの親を「残す」かを指定します。
        通常、親番号は以下の通りです：
        1: 元いたブランチ（Mainline）
        2: マージされたブランチ

 🛠  EXAMPLES
    1. 直前のコミットを取り消す
       $ git revert HEAD
       
    2. マージコミットを取り消す（メインラインを残す）
       $ git revert -m 1 <commit>

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-revert
`
}
