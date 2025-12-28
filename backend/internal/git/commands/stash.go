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
	git.RegisterCommand("stash", func() git.Command { return &StashCommand{} })
}

type StashCommand struct{}

// Ensure StashCommand implements git.Command
var _ git.Command = (*StashCommand)(nil)

const StashRefName = "refs/stash"

func (c *StashCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	op := "push" // default
	if len(args) > 1 {
		op = args[1]
	}

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	switch op {
	case "push", "save":
		return c.executePush(repo, args)
	case "pop":
		return c.executePop(repo)
	case "list":
		return c.executeList(repo)
	case "apply":
		return "", fmt.Errorf("stash apply is not yet supported (use pop)")
	case "drop":
		return "", fmt.Errorf("stash drop is not yet supported")
	default:
		// If arg is not a known subcommand, it might be 'git stash -m "msg"' which implies push
		// For simplicity, treat unknown as push options or error
		return c.executePush(repo, args)
	}
}

func (c *StashCommand) executePush(repo *gogit.Repository, args []string) (string, error) {
	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// 1. Check if there are changes to stash
	status, err := w.Status()
	if err != nil {
		return "", err
	}
	if status.IsClean() {
		return "No local changes to save", nil
	}

	// 2. Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("you do not have the initial commit yet")
	}

	// 3. Resolve Previous Stash (if any) to use as 2nd parent
	var parents []plumbing.Hash
	parents = append(parents, headRef.Hash()) // 1st Parent: HEAD

	prevStashRef, err := repo.Reference(plumbing.ReferenceName(StashRefName), true)
	if err == nil {
		parents = append(parents, prevStashRef.Hash()) // 2nd Parent: Old Stash
	}

	// 4. Create Stash Commit
	// We want to commit the current worktree state.
	// NOTE: Real git stash distinguishes Index vs Worktree. We flatten them for MVP.
	// We add everything to index to capture the snapshot.
	_, err = w.Add(".")
	if err != nil {
		return "", fmt.Errorf("failed to add files for stash: %v", err)
	}

	stashMsg := "WIP on " + headRef.Name().Short() + ": " + time.Now().Format("15:04:05")
	// If User provided a message (e.g. git stash push -m "msg"), parse it?
	// Skipping detailed arg parsing for now.

	stashHash, err := w.Commit(stashMsg, &gogit.CommitOptions{
		Parents: parents,
		Author: &object.Signature{
			Name:  "GitGym Stash",
			Email: "stash@gitgym.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		// Try to rollback index?
		if resetErr := w.Reset(&gogit.ResetOptions{Mode: gogit.MixedReset}); resetErr != nil {
			return "", fmt.Errorf("failed to create stash commit: %v (rollback also failed: %v)", err, resetErr)
		}
		return "", fmt.Errorf("failed to create stash commit: %v", err)
	}

	// 5. Update refs/stash
	stashRef := plumbing.NewHashReference(plumbing.ReferenceName(StashRefName), stashHash)
	err = repo.Storer.SetReference(stashRef)
	if err != nil {
		return "", err
	}

	// 6. Reset Worktree to HEAD (Hard Reset) to clear changes
	err = w.Reset(&gogit.ResetOptions{Mode: gogit.HardReset, Commit: headRef.Hash()})
	if err != nil {
		return "", fmt.Errorf("failed to reset worktree: %v", err)
	}

	return fmt.Sprintf("Saved working directory and index state %s", stashMsg), nil
}

func (c *StashCommand) executePop(repo *gogit.Repository) (string, error) {
	// 1. Resolve Stash
	stashRef, err := repo.Reference(plumbing.ReferenceName(StashRefName), true)
	if err != nil {
		return "No stash entries found.", nil
	}
	stashCommit, err := repo.CommitObject(stashRef.Hash())
	if err != nil {
		return "", err
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	// 2. Apply Stash (Merge)
	// We merge the Stash Commit into current Worktree.
	// Stash Commit has HEAD (at time of stash) as Parent 1.
	// Current HEAD might be different.
	// We use Merge3Way logic: Base=Parent1(OriginalHEAD), Ours=CurrentHEAD, Theirs=StashCommit.

	if stashCommit.NumParents() == 0 {
		return "", fmt.Errorf("invalid stash commit (no parents)")
	}
	baseHash := stashCommit.ParentHashes[0]
	baseCommit, err := repo.CommitObject(baseHash)
	if err != nil {
		return "", fmt.Errorf("could not resolve stash base: %v", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// Attempt Merge
	err = git.Merge3Way(w, baseCommit, headCommit, stashCommit)
	if err != nil {
		if err == git.ErrConflict {
			return "error: conflicts detected during stash pop.\nThe stash was NOT dropped.", nil
		}
		return "", fmt.Errorf("failed to pop stash: %v", err)
	}

	// Merge3Way likely staged the changes (or modified worktree).
	// We want them to be Mixed (unstaged) typically, or Soft (staged).
	// Standard `stash pop` tries to restore index state if `--index` is used, otherwise mixed?
	// Actually `pop` merges changes. If we want them to feel like "work in progress", we usually leave them as modified files.
	// Let's do a Mixed Reset to HEAD (keeping changes in Worktree, but unstaging them).
	// This mimics the "restore work" feel best.
	w.Reset(&gogit.ResetOptions{Mode: gogit.MixedReset})

	// 3. Drop Stash (Move refs/stash to Parent 2)
	// If Parent 2 exists, that is the previous stash.
	if len(stashCommit.ParentHashes) > 1 {
		prevStashHash := stashCommit.ParentHashes[1]
		newRef := plumbing.NewHashReference(plumbing.ReferenceName(StashRefName), prevStashHash)
		repo.Storer.SetReference(newRef)
	} else {
		// Empty stack
		repo.Storer.RemoveReference(plumbing.ReferenceName(StashRefName))
	}

	return fmt.Sprintf("Dropped %s (%s)", StashRefName, stashCommit.Hash.String()[:7]), nil
}

func (c *StashCommand) executeList(repo *gogit.Repository) (string, error) {
	stashRef, err := repo.Reference(plumbing.ReferenceName(StashRefName), true)
	if err != nil {
		return "", nil // Empty list
	}

	var sb strings.Builder

	// Iterate backwards using 2nd parent
	cursor := stashRef.Hash()
	i := 0
	for {
		commit, err := repo.CommitObject(cursor)
		if err != nil {
			break
		}

		// stash@{n}: Message
		sb.WriteString(fmt.Sprintf("stash@{%d}: %s\n", i, strings.TrimSpace(commit.Message)))

		if len(commit.ParentHashes) < 2 {
			break // No more previous stashes
		}
		cursor = commit.ParentHashes[1]
		i++
	}

	return sb.String(), nil
}

func (c *StashCommand) Help() string {
	return `📘 GIT-STASH (1)                                        Git Manual

 💡 DESCRIPTION
    ・作業中の変更（コミットしていない内容）を一時的に退避します。
    ・別のブランチに切り替えたいが、今の作業をコミットしたくない時に使います。

 📋 SYNOPSIS
    git stash [push]
    git stash pop
    git stash list

 🛠  EXAMPLES
    1. 作業を退避する
       $ git stash
       
    2. 退避したリストを見る
       $ git stash list
       
    3. 最新の退避を復元して消す
       $ git stash pop
`
}
