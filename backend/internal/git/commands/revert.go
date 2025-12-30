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

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git revert <commit>")
	}
	// For now, support single commit revert
	rev := args[1]

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
	// If multiple parents (merge commit), fail unless -m is implied (not supported yet)
	if targetCommit.NumParents() > 1 {
		return "", fmt.Errorf("error: commit %s is a merge but no -m option was given", hash.String()[:7])
	}

	var parentCommit *object.Commit
	if targetCommit.NumParents() > 0 {
		parentCommit, err = targetCommit.Parent(0)
		if err != nil {
			return "", err
		}
	} else {
		// Reverting a root commit?
		// "Theirs" should be an empty tree.
		// For simplicity, let's treat it as nil and handle in Merge3Way if it supports it,
		// or create an empty dummy commit if needed.
		// git.Merge3Way usually expects *object.Commit.
		// If implementation supports nil as "empty tree", good. If not, we block root revert for now.
		// Looking at cherry-pick: logic handles Base=nil.
		// Here Parent is "Theirs".
	}

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
    ・すでにPush済みのコミットを取り消す場合に推奨されます。

 📋 SYNOPSIS
    git revert <commit>

 🛠  EXAMPLES
    1. 直前のコミットを取り消す
       $ git revert HEAD
       
    2. 特定の過去のコミットを取り消す
       $ git revert a1b2c3d

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-revert
`
}
