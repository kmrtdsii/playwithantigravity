package commands

import (
	"context"
	"fmt"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("rebase", func() git.Command { return &RebaseCommand{} })
}

type RebaseCommand struct{}

func (c *RebaseCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse Flags & Args
	var (
		onto     string
		upstream string
		branch   string
		preserve bool // -r / --rebase-merges
		root     bool
	)

	// Simple custom parser
	// args[0] is "rebase"
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--onto":
			if i+1 >= len(cmdArgs) {
				return "", fmt.Errorf("fatal: option '--onto' requires a value")
			}
			onto = cmdArgs[i+1]
			i++
		case "-r", "--rebase-merges":
			preserve = true
		case "--root":
			root = true
		case "-h", "--help":
			return c.Help(), nil
		default:
			// Positional arguments: <upstream> [<branch>]
			if upstream == "" {
				upstream = arg
			} else if branch == "" {
				branch = arg
			} else {
				// Too many args? git rebase usually errors or just ignores
				return "", fmt.Errorf("fatal: too many arguments")
			}
		}
	}

	// git rebase --root does not require upstream argument usually
	// usage: git rebase --root [<branch>]
	if upstream == "" && !root && onto == "" {
		return "", fmt.Errorf("usage: git rebase [--onto <newbase>] <upstream> [<branch>]")
	}
	// If only --onto is given without upstream, git usually requires upstream for the range.
	// But `git rebase --onto A B` (B is upstream).
	// Current parser requires positional upstream if not implied?
	// Actually `git rebase --onto newbase upstream` corresponds to `onto=newbase`, `upstream=upstream`.
	// My parser handles this.

	// 1. Checkout Branch if provided
	if branch != "" {
		// "git rebase upstream branch" -> checkout branch then rebase
		// Reuse checkout logic or just simple checkout
		w, _ := repo.Worktree()

		// Try resolving branch to check existence
		hash, err := c.resolveRevision(repo, branch)
		if err != nil {
			return "", fmt.Errorf("fatal: invalid branch '%s'", branch)
		}

		// Checkout
		// Note: We need to update Worktree to point to this branch
		// check if it's a branch name
		refName := plumbing.ReferenceName("refs/heads/" + branch)
		err = w.Checkout(&gogit.CheckoutOptions{
			Branch: refName,
			Force:  true, // Rebase implies we take over? usually strict checkout for working dir safety.
			// But here strict safety might block user. Let's use Force=false first?
			// Actually `git rebase` creates a temporary state.
			// For simulation, let's just switch.
		})
		if err != nil {
			// fallback to detach if not a branch?
			// git rebase <upstream> <commit> triggers detached HEAD rebase.
			err = w.Checkout(&gogit.CheckoutOptions{
				Hash: *hash,
			})
			if err != nil {
				return "", fmt.Errorf("fatal: checkout %s failed: %v", branch, err)
			}
		}
	}

	// Update ORIG_HEAD before rebase starts
	s.UpdateOrigHead()

	// 2. Resolve Upstream (if provided)
	var upstreamHash *plumbing.Hash
	var upstreamCommit *object.Commit

	if upstream != "" {
		h, err := c.resolveRevision(repo, upstream)
		if err != nil {
			return "", fmt.Errorf("invalid upstream '%s': %v", upstream, err)
		}
		upstreamHash = h
		uc, err := repo.CommitObject(*upstreamHash)
		if err != nil {
			return "", err
		}
		upstreamCommit = uc
	} else if !root {
		return "", fmt.Errorf("fatal: upstream required")
	}

	// 3. Resolve NewBase (target)
	// If --onto is specified, we reset to 'onto'. Otherwise we reset to 'upstream'.
	var targetHash *plumbing.Hash
	if onto != "" {
		h, err := c.resolveRevision(repo, onto)
		if err != nil {
			return "", fmt.Errorf("invalid onto '%s': %v", onto, err)
		}
		targetHash = h
	} else {
		// If --onto not specified, reset to upstream
		// If --root is specified without --onto, we are just re-writing history in place?
		// "git rebase --root" typically replays all history on top of... itself?
		// Effectively used to squash root or something.
		// "git rebase --root <branch>" defaults to replaying on top of... nothing (orphan)?
		// No, `git rebase --root` without `onto` is usually for interactive mode or squashing.
		// For simple rebase, if --root is given without --onto,
		// "The new base is the first commit in the sequence." - wait.

		if root && upstreamHash == nil {
			// If no upstream and no onto, we probably shouldn't be here in this simple implem unless interactive.
			// But let's assume if upstream IS provided with --root, it might be ignored or used?
			// Git docs: "git rebase --root <branch>" -> rewrite all commits reachable from <branch>.
			// Default base?
			// For now, require --onto for --root usage to be safe/simple.
			return "", fmt.Errorf("fatal: --onto required with --root for now")
		}
		targetHash = upstreamHash
	}

	// 4. Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	// 5. Find Commits to Replay
	var commitsToReplay []*object.Commit
	var base *object.Commit

	if root {
		// Replay ALL reachable commits from HEAD (down to root)
		// We ignore upstream for calculation of range (upstream..HEAD]
		// We just traverse down to root.
		iter := headCommit
		for {
			commitsToReplay = append(commitsToReplay, iter)
			if iter.NumParents() == 0 {
				break
			}
			p, pErr := iter.Parent(0)
			if pErr != nil {
				return "", fmt.Errorf("failed to traverse parents: %v", pErr)
			}
			iter = p
		}
	} else {
		// Standard upstream..HEAD calculation
		mergeBases, mbErr := upstreamCommit.MergeBase(headCommit)
		if mbErr != nil {
			return "", fmt.Errorf("failed to find merge base: %v", mbErr)
		}
		if len(mergeBases) == 0 {
			return "", fmt.Errorf("fatal: no common ancestor found. Use --root to rebase unrelated histories.")
		}
		base = mergeBases[0]

		// Optimization: If HEAD is already based on Target (and no commits to replay?), we are done?
		// But if --onto is used, we definitely move.
		// If NOT --onto, and HEAD is already descendant of Upstream, we are "up to date".

		if onto == "" {
			if base.Hash == headCommit.Hash {
				// HEAD is ancestor of Upstream? Then Fast-Forward or we are behind.
				// "Current branch is up to date" usually means HEAD == Upstream
				// If HEAD is behind, rebase usually fast-forwards HEAD to upstream.
				// Let's assume FF.
				// .. logic ..
				_ = base
			}
			if base.Hash == upstreamCommit.Hash {
				// Upstream is ancestor of HEAD. We are ahead.
				// If we rebase onto upstream, nothing changes locally (already on top).
				return "Current branch is up to date.", nil
			}
		}

		// Collect commits (Base..HEAD]
		var iter = headCommit
		for iter.Hash != base.Hash {
			// Safety check to avoid infinite loop if detached or disjoint
			commitsToReplay = append(commitsToReplay, iter)
			if iter.NumParents() == 0 {
				break
			}
			p, pErr := iter.Parent(0) // Linear history assumption for simple rebase
			if pErr != nil {
				return "", fmt.Errorf("failed to traverse parents: %v", pErr)
			}
			iter = p
		}
	}

	// Reverse to replay oldest first
	for i, j := 0, len(commitsToReplay)-1; i < j; i, j = i+1, j-1 {
		commitsToReplay[i], commitsToReplay[j] = commitsToReplay[j], commitsToReplay[i]
	}

	// 6. Hard Reset to Target (NewBase)
	w, _ := repo.Worktree()
	if resetErr := w.Reset(&gogit.ResetOptions{Commit: *targetHash, Mode: gogit.HardReset}); resetErr != nil {
		return "", fmt.Errorf("failed to reset to newbase: %v", resetErr)
	}

	// 7. Replay Commits (using shared helper)
	replayedCount := 0
	for _, c := range commitsToReplay {
		// Apply changes from this commit using shared helper
		if applyErr := git.ApplyCommitChanges(w, c); applyErr != nil {
			return "", fmt.Errorf("failed to apply commit %s: %v", c.Hash.String()[:7], applyErr)
		}

		// Ensure timestamp distinctness
		time.Sleep(10 * time.Millisecond)

		_, err = w.Commit(c.Message, &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
			AllowEmptyCommits: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to commit replayed change: %v", err)
		}
		replayedCount++
	}

	if preserve {
		// We aren't actually implementing merge preservation yet, just acknowledged flag.
		_ = preserve
	}

	s.RecordReflog(fmt.Sprintf("rebase: finished rebase onto %s", targetHash.String()))
	return fmt.Sprintf("Successfully rebased and updated %s.\nReplayed %d commits.", headRef.Name().Short(), replayedCount), nil
}

// resolveRevision delegates to the shared git.ResolveRevision helper
func (c *RebaseCommand) resolveRevision(repo *gogit.Repository, rev string) (*plumbing.Hash, error) {
	return git.ResolveRevision(repo, rev)
}

func (c *RebaseCommand) Help() string {
	return `usage: git rebase [-r] [--onto <newbase>] <upstream> [<branch>]

Reapply commits on top of another base tip.
`
}
