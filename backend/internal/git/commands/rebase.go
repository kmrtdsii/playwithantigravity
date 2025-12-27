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
	git.RegisterCommand("rebase", func() git.Command { return &RebaseCommand{} })
}

type RebaseCommand struct{}

type RebaseOptions struct {
	Upstream string
	Branch   string
	Onto     string
	Root     bool
	Preserve bool
}

type rebaseContext struct {
	targetHash      *plumbing.Hash
	commitsToReplay []*object.Commit
	headRef         *plumbing.Reference // Needed for success message
}

func (c *RebaseCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	// 2. Checkout Branch if provided
	if opts.Branch != "" {
		if err := c.checkoutBranch(repo, opts.Branch); err != nil {
			return "", err
		}
	}

	// Update ORIG_HEAD before rebase starts
	s.UpdateOrigHead()

	// 3. Prepare Rebase Context (resolve revisions, find commits)
	rbCtx, err := c.prepareRebaseContext(repo, opts)
	if err != nil {
		if err == ErrUpToDate {
			return "Current branch is up to date.", nil
		}
		return "", err
	}

	// 4. Perform Rebase
	return c.performRebase(ctx, s, repo, rbCtx, opts.Preserve)
}

var ErrUpToDate = fmt.Errorf("up to date")

func (c *RebaseCommand) parseArgs(args []string) (*RebaseOptions, error) {
	opts := &RebaseOptions{}
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "--onto":
			if i+1 >= len(cmdArgs) {
				return nil, fmt.Errorf("fatal: option '--onto' requires a value")
			}
			opts.Onto = cmdArgs[i+1]
			i++
		case "-r", "--rebase-merges":
			opts.Preserve = true
		case "--root":
			opts.Root = true
		case "-h", "--help":
			// Handled by calling Help() at higher level usually, but here checking arg
			return nil, fmt.Errorf("help requested") // Should effectively show help if strictly followed, but standard is different. Logic in Execute handles it? No, Execute returns string/error.
			// Let's assume help is handled before Execute or if we return specific error?
			// Existing logic called c.Help() directly.
		default:
			if strings.HasPrefix(arg, "-") {
				continue // ignore unknown flags
			}
			if opts.Upstream == "" {
				opts.Upstream = arg
			} else if opts.Branch == "" {
				opts.Branch = arg
			} else {
				return nil, fmt.Errorf("fatal: too many arguments")
			}
		}
	}

	if opts.Upstream == "" && !opts.Root && opts.Onto == "" {
		return nil, fmt.Errorf("usage: git rebase [--onto <newbase>] <upstream> [<branch>]")
	}
	return opts, nil
}

func (c *RebaseCommand) checkoutBranch(repo *gogit.Repository, branchName string) error {
	w, _ := repo.Worktree()
	hash, err := git.ResolveRevision(repo, branchName)
	if err != nil {
		return fmt.Errorf("fatal: invalid branch '%s'", branchName)
	}

	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	err = w.Checkout(&gogit.CheckoutOptions{
		Branch: refName,
		Force:  true,
	})
	if err != nil {
		// fallback to detach
		err = w.Checkout(&gogit.CheckoutOptions{
			Hash: *hash,
		})
		if err != nil {
			return fmt.Errorf("fatal: checkout %s failed: %v", branchName, err)
		}
	}
	return nil
}

func (c *RebaseCommand) prepareRebaseContext(repo *gogit.Repository, opts *RebaseOptions) (*rebaseContext, error) {
	// Resolve Upstream
	var upstreamHash *plumbing.Hash
	var upstreamCommit *object.Commit

	if opts.Upstream != "" {
		h, err := git.ResolveRevision(repo, opts.Upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream '%s': %v", opts.Upstream, err)
		}
		upstreamHash = h
		uc, err := repo.CommitObject(*upstreamHash)
		if err != nil {
			return nil, err
		}
		upstreamCommit = uc
	} else if !opts.Root {
		return nil, fmt.Errorf("fatal: upstream required")
	}

	// Resolve NewBase (target)
	var targetHash *plumbing.Hash
	if opts.Onto != "" {
		h, err := git.ResolveRevision(repo, opts.Onto)
		if err != nil {
			return nil, fmt.Errorf("invalid onto '%s': %v", opts.Onto, err)
		}
		targetHash = h
	} else {
		if opts.Root && upstreamHash == nil {
			return nil, fmt.Errorf("fatal: --onto required with --root for now")
		}
		targetHash = upstreamHash
	}

	// Resolve HEAD
	headRef, err := repo.Head()
	if err != nil {
		return nil, err
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, err
	}

	// Find Commits to Replay
	var commitsToReplay []*object.Commit

	if opts.Root {
		// Replay ALL reachable commits from HEAD (down to root)
		iter := headCommit
		for {
			commitsToReplay = append(commitsToReplay, iter)
			if iter.NumParents() == 0 {
				break
			}
			p, pErr := iter.Parent(0)
			if pErr != nil {
				return nil, fmt.Errorf("failed to traverse parents: %v", pErr)
			}
			iter = p
		}
	} else {
		// Standard upstream..HEAD calculation
		mergeBases, mbErr := upstreamCommit.MergeBase(headCommit)
		if mbErr != nil {
			return nil, fmt.Errorf("failed to find merge base: %v", mbErr)
		}
		if len(mergeBases) == 0 {
			return nil, fmt.Errorf("fatal: no common ancestor found. Use --root to rebase unrelated histories.")
		}
		base := mergeBases[0]

		// Check for up-to-date
		if opts.Onto == "" {
			if base.Hash == upstreamCommit.Hash {
				return nil, ErrUpToDate
			}
		}

		// Collect commits (Base..HEAD]
		var iter = headCommit
		for iter.Hash != base.Hash {
			commitsToReplay = append(commitsToReplay, iter)
			if iter.NumParents() == 0 {
				break
			}
			p, pErr := iter.Parent(0)
			if pErr != nil {
				return nil, fmt.Errorf("failed to traverse parents: %v", pErr)
			}
			iter = p
		}
	}

	// Reverse to replay oldest first
	for i, j := 0, len(commitsToReplay)-1; i < j; i, j = i+1, j-1 {
		commitsToReplay[i], commitsToReplay[j] = commitsToReplay[j], commitsToReplay[i]
	}

	return &rebaseContext{
		targetHash:      targetHash,
		commitsToReplay: commitsToReplay,
		headRef:         headRef,
	}, nil
}

func (c *RebaseCommand) performRebase(ctx context.Context, s *git.Session, repo *gogit.Repository, rbCtx *rebaseContext, preserve bool) (string, error) {
	// Hard Reset to Target (NewBase)
	w, _ := repo.Worktree()
	if resetErr := w.Reset(&gogit.ResetOptions{Commit: *rbCtx.targetHash, Mode: gogit.HardReset}); resetErr != nil {
		return "", fmt.Errorf("failed to reset to newbase: %v", resetErr)
	}

	// Replay Commits
	replayedCount := 0
	for _, c := range rbCtx.commitsToReplay {
		if applyErr := git.ApplyCommitChanges(w, c); applyErr != nil {
			return "", fmt.Errorf("failed to apply commit %s: %v", c.Hash.String()[:7], applyErr)
		}

		// Ensure timestamp distinctness
		time.Sleep(10 * time.Millisecond)

		_, err := w.Commit(c.Message, &gogit.CommitOptions{
			Author:            git.GetDefaultSignature(),
			AllowEmptyCommits: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to commit replayed change: %v", err)
		}
		replayedCount++
	}

	s.RecordReflog(fmt.Sprintf("rebase: finished rebase onto %s", rbCtx.targetHash.String()))
	return fmt.Sprintf("Successfully rebased and updated %s.\nReplayed %d commits.", rbCtx.headRef.Name().Short(), replayedCount), nil
}

func (c *RebaseCommand) Help() string {
	return `📘 GIT-REBASE (1)                                       Git Manual

 💡 DESCRIPTION
    ・ブランチの派生元（親コミット）を付け替える
    ・コミットの履歴を整形して、一直線にする
    （歴史を書き換えるため、共有されているブランチでの使用は注意が必要です）
    
    ⚠️ 注意: 既に公開（プッシュ）したコミットをリベースすることは推奨されません。

 📋 SYNOPSIS
    git rebase [--onto <newbase>] <upstream> [<branch>]
    git rebase --root

 ⚙️  COMMON OPTIONS
    --onto <newbase>
        新しいベース地点を明示的に指定します。

    --root
        ルートコミット（最初のコミット）まで遡ってリベースします。

 🛠  EXAMPLES
    1. 現在のブランチをmainの最新に追従させる
       $ git rebase main
`
}
