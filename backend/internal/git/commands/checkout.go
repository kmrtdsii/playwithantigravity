package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/kurobon/gitgym/backend/internal/git/commands/checkout"
)

func init() {
	git.RegisterCommand("checkout", func() git.Command { return &CheckoutCommand{} })
}

// CheckoutCommand implements the git checkout command.
type CheckoutCommand struct{}

// Ensure CheckoutCommand implements git.Command
var _ git.Command = (*CheckoutCommand)(nil)

// Strategy instances (stateless, can be shared)
var (
	fileStrategy   = &checkout.FileStrategy{}
	orphanStrategy = &checkout.OrphanStrategy{}
	branchStrategy = &checkout.BranchStrategy{}
	refStrategy    = &checkout.RefStrategy{}
)

func (c *CheckoutCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	// 2. Resolve Context
	cCtx, err := c.resolveContext(repo, opts)
	if err != nil {
		return "", err
	}

	// 3. Dispatch to Strategy
	strategy := c.selectStrategy(cCtx.Mode)
	if strategy == nil {
		return "", fmt.Errorf("internal error: unknown checkout mode")
	}
	return strategy.Execute(s, cCtx, opts)
}

func (c *CheckoutCommand) selectStrategy(mode checkout.Mode) checkout.Strategy {
	switch mode {
	case checkout.ModeFiles:
		return fileStrategy
	case checkout.ModeOrphan:
		return orphanStrategy
	case checkout.ModeNewBranch:
		return branchStrategy
	case checkout.ModeRefOrPath:
		return refStrategy
	default:
		return nil
	}
}

func (c *CheckoutCommand) parseArgs(args []string) (*checkout.Options, error) {
	opts := &checkout.Options{}
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-b":
			if i+1 >= len(cmdArgs) {
				return nil, fmt.Errorf("fatal: missing branch name for -b")
			}
			opts.NewBranch = cmdArgs[i+1]
			i++
		case "-B":
			if i+1 >= len(cmdArgs) {
				return nil, fmt.Errorf("fatal: missing branch name for -B")
			}
			opts.ForceNewBranch = cmdArgs[i+1]
			i++
		case "--orphan":
			if i+1 >= len(cmdArgs) {
				return nil, fmt.Errorf("fatal: missing branch name for --orphan")
			}
			opts.OrphanBranch = cmdArgs[i+1]
			i++
		case "-f", "--force":
			opts.Force = true
		case "--detach":
			opts.Detach = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		case "--":
			// End of flags, remainder are paths
			if i+1 >= len(cmdArgs) {
				return nil, fmt.Errorf("fatal: filename required after --")
			}
			opts.Files = cmdArgs[i+1:]
			return opts, nil // Return immediately as loose args are consumed
		default:
			if opts.Target == "" {
				opts.Target = arg
			}
		}
	}
	return opts, nil
}

func (c *CheckoutCommand) resolveContext(repo *gogit.Repository, opts *checkout.Options) (*checkout.Context, error) {
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	ctx := &checkout.Context{
		Worktree: w,
		Repo:     repo,
	}

	// Determine Mode
	if len(opts.Files) > 0 {
		ctx.Mode = checkout.ModeFiles
		ctx.Files = opts.Files
		return ctx, nil
	}

	if opts.OrphanBranch != "" {
		ctx.Mode = checkout.ModeOrphan
		ctx.OrphanBranch = opts.OrphanBranch

		// Verify it doesn't exist
		refName := plumbing.ReferenceName("refs/heads/" + opts.OrphanBranch)
		_, err := repo.Reference(refName, true)
		if err == nil {
			return nil, fmt.Errorf("fatal: a branch named '%s' already exists", opts.OrphanBranch)
		}
		return ctx, nil
	}

	if opts.NewBranch != "" || opts.ForceNewBranch != "" {
		ctx.Mode = checkout.ModeNewBranch
		ctx.NewBranch = opts.NewBranch
		ctx.ForceCreate = false
		if opts.ForceNewBranch != "" {
			ctx.NewBranch = opts.ForceNewBranch
			ctx.ForceCreate = true
		}

		startPoint := opts.Target
		if startPoint == "" {
			startPoint = "HEAD"
		}

		hash, err := repo.ResolveRevision(plumbing.Revision(startPoint))
		if err != nil {
			return nil, fmt.Errorf("fatal: invalid reference: %s", startPoint)
		}
		ctx.StartPointHash = hash

		refName := plumbing.ReferenceName("refs/heads/" + ctx.NewBranch)
		_, err = repo.Reference(refName, true)
		if err == nil && !ctx.ForceCreate {
			return nil, fmt.Errorf("fatal: a branch named '%s' already exists", ctx.NewBranch)
		}
		return ctx, nil
	}

	if opts.Target == "" {
		return nil, fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch>")
	}

	// Ref or Path mode
	ctx.Mode = checkout.ModeRefOrPath

	// 1. Try as branch (unless --detach)
	if !opts.Detach {
		branchRef := plumbing.ReferenceName("refs/heads/" + opts.Target)
		_, err := repo.Reference(branchRef, true)
		if err == nil {
			ctx.TargetRef = branchRef
			return ctx, nil
		}
	}

	// 1.5. Check if it's a remote branch (Auto-track)
	remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", opts.Target))
	if remoteRef, err := repo.Reference(remoteRefName, true); err == nil && !opts.Detach {
		ctx.TargetRef = remoteRefName
		h := remoteRef.Hash()
		ctx.TargetHash = &h
		return ctx, nil
	}

	// 2. Try as hash/tag (Detached HEAD)
	hash, err := repo.ResolveRevision(plumbing.Revision(opts.Target))
	if err == nil {
		if _, errObj := repo.CommitObject(*hash); errObj == nil { // is commit
			ctx.TargetHash = hash
			ctx.IsDetached = true
			return ctx, nil
		}
	}

	// 3. Fallback: treat as file path?
	// Check if file exists in HEAD
	headRef, err := repo.Head()
	if err == nil {
		headCommit, err := repo.CommitObject(headRef.Hash())
		if err == nil {
			if _, errFile := headCommit.File(opts.Target); errFile == nil {
				ctx.Mode = checkout.ModeFiles
				ctx.Files = []string{opts.Target}
				return ctx, nil
			}
		}
	}

	return nil, fmt.Errorf("error: pathspec '%s' did not match any file(s) known to git", opts.Target)
}

func (c *CheckoutCommand) Help() string {
	return `📘 GIT-CHECKOUT (1)                                     Git Manual

 💡 DESCRIPTION
    HEAD（今作業しているブランチやコミット）を移動します。
    それに合わせて、手元のファイル（ワーキングツリー）の内容も更新されます。
    
    主な用途：
    1. 他のブランチに切り替える（switch使おう）
    2. 新しいブランチを作って切り替える（switch -c使おう）
    3. ファイルの変更を取り消して元に戻す（この使い方が重要！）

 📋 SYNOPSIS
    git checkout <branch>
    git checkout -b <new_branch>
    git checkout -- <file>...

 ⚙️  COMMON OPTIONS
    -b <new_branch>
        新しいブランチを作成して、すぐにそのブランチに切り替えます。

    -B <new_branch>
        ブランチが存在しても強制的に作成（リセット）して切り替えます。
    
    -- <file>
        ブランチ切り替えではなく、指定したファイルの変更を取り消して元に戻します。

 🛠  PRACTICAL EXAMPLES
    1. 基本: 既存のブランチに切り替え
       $ git checkout main

    2. 基本: 新しいブランチを作成して切り替え
       $ git checkout -b feature/login

    3. 実践: 変更の取り消し (Important)
       「コードいじってたら動かなくなった...元に戻したい」
       そんな時は、ファイルを指定して checkout します。
       $ git checkout -- src/main.go

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-checkout
`
}
