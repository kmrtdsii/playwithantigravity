package commands

import (
	"context"
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("checkout", func() git.Command { return &CheckoutCommand{} })
}

type CheckoutCommand struct{}

// Ensure CheckoutCommand implements git.Command
var _ git.Command = (*CheckoutCommand)(nil)

type CheckoutOptions struct {
	NewBranch      string
	ForceNewBranch string
	OrphanBranch   string
	Force          bool
	Detach         bool
	Target         string
	Files          []string // For "git checkout -- <file>"
}

type checkoutMode int

const (
	modeInvalid checkoutMode = iota
	modeFiles
	modeOrphan
	modeNewBranch
	modeRefOrPath
)

type checkoutContext struct {
	mode           checkoutMode
	w              *gogit.Worktree
	repo           *gogit.Repository
	files          []string
	orphanBranch   string
	newBranch      string
	forceCreate    bool
	startPointHash *plumbing.Hash
	targetRef      plumbing.ReferenceName
	targetHash     *plumbing.Hash
	isDetached     bool
}

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

	// 2. Resolve
	cCtx, err := c.resolveContext(repo, opts)
	if err != nil {
		return "", err
	}

	// 3. Perform
	return c.performAction(s, cCtx, opts)
}

func (c *CheckoutCommand) parseArgs(args []string) (*CheckoutOptions, error) {
	opts := &CheckoutOptions{}
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

func (c *CheckoutCommand) resolveContext(repo *gogit.Repository, opts *CheckoutOptions) (*checkoutContext, error) {
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	ctx := &checkoutContext{
		w:    w,
		repo: repo,
	}

	// Determine Mode
	if len(opts.Files) > 0 {
		ctx.mode = modeFiles
		ctx.files = opts.Files
		return ctx, nil
	}

	if opts.OrphanBranch != "" {
		ctx.mode = modeOrphan
		ctx.orphanBranch = opts.OrphanBranch

		// Verify it doesn't exist
		refName := plumbing.ReferenceName("refs/heads/" + opts.OrphanBranch)
		_, err := repo.Reference(refName, true)
		if err == nil {
			return nil, fmt.Errorf("fatal: a branch named '%s' already exists", opts.OrphanBranch)
		}
		return ctx, nil
	}

	if opts.NewBranch != "" || opts.ForceNewBranch != "" {
		ctx.mode = modeNewBranch
		ctx.newBranch = opts.NewBranch
		ctx.forceCreate = false
		if opts.ForceNewBranch != "" {
			ctx.newBranch = opts.ForceNewBranch
			ctx.forceCreate = true
		}

		startPoint := opts.Target
		if startPoint == "" {
			startPoint = "HEAD"
		}

		hash, err := repo.ResolveRevision(plumbing.Revision(startPoint))
		if err != nil {
			return nil, fmt.Errorf("fatal: invalid reference: %s", startPoint)
		}
		ctx.startPointHash = hash

		refName := plumbing.ReferenceName("refs/heads/" + ctx.newBranch)
		_, err = repo.Reference(refName, true)
		if err == nil && !ctx.forceCreate {
			return nil, fmt.Errorf("fatal: a branch named '%s' already exists", ctx.newBranch)
		}
		return ctx, nil
	}

	if opts.Target == "" {
		return nil, fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch>")
	}

	// Ref or Path mode
	ctx.mode = modeRefOrPath

	// 1. Try as branch (unless --detach)
	if !opts.Detach {
		branchRef := plumbing.ReferenceName("refs/heads/" + opts.Target)
		_, err := repo.Reference(branchRef, true)
		if err == nil {
			ctx.targetRef = branchRef
			return ctx, nil
		}
	}

	// 1.5. Check if it's a remote branch (Auto-track)
	remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", opts.Target))
	if remoteRef, err := repo.Reference(remoteRefName, true); err == nil && !opts.Detach {
		ctx.targetRef = remoteRefName
		h := remoteRef.Hash()
		ctx.targetHash = &h
		return ctx, nil
	}

	// 2. Try as hash/tag (Detached HEAD)
	hash, err := repo.ResolveRevision(plumbing.Revision(opts.Target))
	if err == nil {
		if _, errObj := repo.CommitObject(*hash); errObj == nil { // is commit
			ctx.targetHash = hash
			ctx.isDetached = true
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
				ctx.mode = modeFiles
				ctx.files = []string{opts.Target}
				return ctx, nil
			}
		}
	}

	return nil, fmt.Errorf("error: pathspec '%s' did not match any file(s) known to git", opts.Target)
}

func (c *CheckoutCommand) performAction(s *git.Session, ctx *checkoutContext, opts *CheckoutOptions) (string, error) {
	switch ctx.mode {
	case modeFiles:
		return c.executeCheckoutFiles(ctx.repo, ctx.w, ctx.files)
	case modeOrphan:
		return c.executeCheckoutOrphan(ctx.repo, s, ctx.orphanBranch)
	case modeNewBranch:
		return c.executeCreateAndCheckout(ctx.repo, ctx.w, s, ctx.newBranch, ctx.startPointHash, ctx.forceCreate, opts.Force)
	case modeRefOrPath:
		return c.executeCheckoutRef(ctx.repo, ctx.w, s, opts.Target, ctx.targetRef, ctx.targetHash, opts.Force, ctx.isDetached)
	default:
		return "", fmt.Errorf("internal error: unknown checkout mode")
	}
}

func (c *CheckoutCommand) executeCheckoutOrphan(repo *gogit.Repository, s *git.Session, branchName string) (string, error) {
	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
	if err := repo.Storer.SetReference(headRef); err != nil {
		return "", fmt.Errorf("failed to set HEAD for orphan: %w", err)
	}

	s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s (orphan)", "HEAD", branchName))
	return fmt.Sprintf("Switched to a new branch '%s' (orphan)", branchName), nil
}

func (c *CheckoutCommand) executeCheckoutFiles(repo *gogit.Repository, w *gogit.Worktree, files []string) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("fatal: cannot checkout file without HEAD")
	}
	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", err
	}

	for _, filename := range files {
		file, err := headCommit.File(filename)
		if err != nil {
			return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", filename)
		}
		content, _ := file.Contents()

		f, err := w.Filesystem.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return "", err
		}
		_, _ = f.Write([]byte(content))
		_ = f.Close()
	}

	if len(files) == 1 {
		return "Updated " + files[0], nil
	}
	return fmt.Sprintf("Updated %d files", len(files)), nil
}

func (c *CheckoutCommand) executeCreateAndCheckout(repo *gogit.Repository, w *gogit.Worktree, s *git.Session, branchName string, hash *plumbing.Hash, forceCreate, forceCheckout bool) (string, error) {
	refName := plumbing.ReferenceName("refs/heads/" + branchName)
	newRef := plumbing.NewHashReference(refName, *hash)
	if err := repo.Storer.SetReference(newRef); err != nil {
		return "", err
	}

	err := w.Checkout(&gogit.CheckoutOptions{
		Branch: refName,
		Force:  forceCheckout,
	})
	if err != nil {
		return "", err
	}

	s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
	if forceCreate {
		return fmt.Sprintf("Reset branch '%s'", branchName), nil
	}
	return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
}

func (c *CheckoutCommand) executeCheckoutRef(repo *gogit.Repository, w *gogit.Worktree, s *git.Session, targetName string, ref plumbing.ReferenceName, hash *plumbing.Hash, force, detach bool) (string, error) {
	gOpts := &gogit.CheckoutOptions{Force: force}

	if ref != "" {
		if ref.IsRemote() {
			// Actually need to create local if it's remote auto-track
			localName := targetName
			localRef := plumbing.ReferenceName("refs/heads/" + localName)
			newRef := plumbing.NewHashReference(localRef, *hash)
			if err := repo.Storer.SetReference(newRef); err != nil {
				return "", err
			}
			gOpts.Branch = localRef
		} else {
			gOpts.Branch = ref
		}
	} else if hash != nil {
		gOpts.Hash = *hash
	}

	if err := w.Checkout(gOpts); err != nil {
		return "", err
	}

	s.RecordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", targetName))

	if detach {
		return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", targetName), nil
	}
	if ref != "" && ref.IsRemote() {
		return fmt.Sprintf("Switched to a new branch '%s'\nBranch '%s' set up to track remote branch '%s' from 'origin'.", targetName, targetName, targetName), nil
	}
	return fmt.Sprintf("Switched to branch '%s'", targetName), nil
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
