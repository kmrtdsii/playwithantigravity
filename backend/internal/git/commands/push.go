package commands

// push.go - Simulated Git Push Command
//
// IMPORTANT: This implementation does NOT perform actual network operations.
// It simulates push by copying objects between in-memory repositories:
//   - Session-local repos (s.Repos)
//   - Shared virtual remotes (s.Manager.SharedRemotes)
//
// This is safe for educational/sandbox use. No real GitHub/GitLab push occurs.

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("push", func() git.Command { return &PushCommand{} })
}

type PushCommand struct{}

// Ensure PushCommand implements git.Command
var _ git.Command = (*PushCommand)(nil)

type PushOptions struct {
	Remote  string
	Refspec string
	Force   bool
	DryRun  bool
}

type pushContext struct {
	TargetRepo *gogit.Repository
	RemoteName string
	RemoteURL  string
	Ref        *plumbing.Reference // The local ref to push (HEAD or specific branch/tag)
}

func (c *PushCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Parse Args
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	if opts.DryRun {
		// Quick check before resolution? Or resolved dry run?
		// Logic below does full resolution then prints matches.
	}

	// 2. Resolve Context (Remote, TargetRepo, RefToPush)
	pCtx, err := c.resolveContext(s, repo, opts)
	if err != nil {
		return "", err
	}

	// 3. Execution (Perform Push)
	return c.performPush(repo, pCtx, opts)
}

func (c *PushCommand) parseArgs(args []string) (*PushOptions, error) {
	opts := &PushOptions{
		Remote: "origin", // Default
	}
	var positional []string

	cmdArgs := args[1:]
	for _, arg := range cmdArgs {
		switch arg {
		case "-f", "--force":
			opts.Force = true
		case "-n", "--dry-run":
			opts.DryRun = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if strings.HasPrefix(arg, "-") {
				// Ignore unknown flags or error? Legacy ignored.
				// Let's stick to simple "ignore" or strict?
				// Legacy: switch default: if strings.HasPrefix(arg, "-") { ignore } else { positional... }
			} else {
				positional = append(positional, arg)
			}
		}
	}

	if len(positional) > 0 {
		opts.Remote = positional[0]
	}
	if len(positional) > 1 {
		opts.Refspec = positional[1]
	}

	return opts, nil
}

func (c *PushCommand) resolveContext(s *git.Session, repo *gogit.Repository, opts *PushOptions) (*pushContext, error) {
	// Resolve Remote URL
	rem, err := repo.Remote(opts.Remote)
	if err != nil {
		return nil, fmt.Errorf("fatal: '%s' does not appear to be a git repository", opts.Remote)
	}

	cfg := rem.Config()
	if len(cfg.URLs) == 0 {
		return nil, fmt.Errorf("remote %s has no URL defined", opts.Remote)
	}
	url := cfg.URLs[0]

	// Resolve local simulated remote path
	lookupKey := strings.TrimPrefix(url, "/")

	var targetRepo *gogit.Repository
	var ok bool

	// Check Session-local Repos
	targetRepo, ok = s.Repos[lookupKey]
	if !ok && s.Manager != nil {
		// Check Shared Remotes
		targetRepo, ok = s.Manager.SharedRemotes[lookupKey] // e.g. "repo.git"

		// Fallback: Check using full URL
		if !ok {
			targetRepo, ok = s.Manager.SharedRemotes[url]
		}
	}

	if !ok {
		// FALLBACK: Local filesystem path (persistent remote)
		targetRepo, err = gogit.PlainOpen(url)
		if err == nil {
			ok = true
		} else {
			targetRepo, err = gogit.PlainOpen(lookupKey)
			if err == nil {
				ok = true
			}
		}
	}

	if !ok {
		return nil, fmt.Errorf("remote repository '%s' not found (only local simulation supported)", url)
	}

	// Determined Ref to Push
	var refToPush *plumbing.Reference

	if opts.Refspec != "" {
		// Try to resolve refspec (Branch or Tag)
		ref, refErr := repo.Reference(plumbing.ReferenceName(opts.Refspec), true)
		if refErr == nil {
			refToPush = ref
		} else {
			// Try refs/heads/
			ref, err = repo.Reference(plumbing.ReferenceName("refs/heads/"+opts.Refspec), true)
			if err == nil {
				refToPush = ref
			} else {
				// Try refs/tags/
				ref, err = repo.Reference(plumbing.ReferenceName("refs/tags/"+opts.Refspec), true)
				if err == nil {
					refToPush = ref
				} else {
					return nil, fmt.Errorf("src refspec '%s' does not match any", opts.Refspec)
				}
			}
		}
	} else {
		// Default: Push HEAD
		headRef, headErr := repo.Head()
		if headErr != nil {
			return nil, fmt.Errorf("failed to get HEAD: %w", headErr)
		}
		if !headRef.Name().IsBranch() {
			return nil, fmt.Errorf("HEAD is not on a branch (detached?)")
		}
		refToPush = headRef
	}

	return &pushContext{
		TargetRepo: targetRepo,
		RemoteName: opts.Remote,
		RemoteURL:  url,
		Ref:        refToPush,
	}, nil
}

func (c *PushCommand) performPush(repo *gogit.Repository, pCtx *pushContext, opts *PushOptions) (string, error) {
	refName := pCtx.Ref.Name()
	targetRepo := pCtx.TargetRepo

	// Check Fast-Forward (only for branches)
	if refName.IsBranch() && !opts.Force {
		targetRef, targetErr := targetRepo.Reference(refName, true)
		if targetErr == nil {
			isFF, gitErr := git.IsFastForward(repo, targetRef.Hash(), pCtx.Ref.Hash())
			if gitErr != nil {
				return "", gitErr
			}
			if !isFF {
				return "", fmt.Errorf("non-fast-forward update rejected (use --force to override)")
			}
		}
	} else if refName.IsTag() {
		_, tagRefErr := targetRepo.Reference(refName, true)
		if tagRefErr == nil && !opts.Force {
			return "", fmt.Errorf("tag '%s' already exists (use --force to override)", refName.Short())
		}
	}

	if opts.DryRun {
		return fmt.Sprintf("[dry-run] Would push %s to %s at %s", refName.Short(), pCtx.RemoteName, pCtx.RemoteURL), nil
	}

	// SIMULATE PUSH: Copy Objects + Update Ref
	hashToSync := pCtx.Ref.Hash()

	// Check object type
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hashToSync)
	if err != nil {
		return "", err
	}

	if obj.Type() == plumbing.TagObject {
		// Annotated tag logic
		if !git.HasObject(targetRepo, hashToSync) {
			_, err = targetRepo.Storer.SetEncodedObject(obj)
			if err != nil {
				return "", err
			}
		}
		// Decode tag to find target commit
		tagObj, decodeErr := object.DecodeTag(repo.Storer, obj)
		if decodeErr != nil {
			return "", decodeErr
		}
		if copyErr := git.CopyCommitRecursive(repo, targetRepo, tagObj.Target); copyErr != nil {
			return "", copyErr
		}
	} else if obj.Type() == plumbing.CommitObject {
		if copyErr := git.CopyCommitRecursive(repo, targetRepo, hashToSync); copyErr != nil {
			return "", copyErr
		}
	} else {
		return "", fmt.Errorf("unsupported object type to push: %s", obj.Type())
	}

	// Update Remote Reference
	err = targetRepo.Storer.SetReference(pCtx.Ref)
	if err != nil {
		return "", err
	}

	// Update Local Remote-Tracking Reference (ONLY for branches)
	if refName.IsBranch() {
		localRemoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", pCtx.RemoteName, refName.Short()))
		newLocalRemoteRef := plumbing.NewHashReference(localRemoteRefName, hashToSync)
		_ = repo.Storer.SetReference(newLocalRemoteRef)
	}

	return fmt.Sprintf("To %s\n   %s -> %s", pCtx.RemoteURL, hashToSync.String()[:7], refName.Short()), nil
}

func (c *PushCommand) Help() string {
	return `📘 GIT-PUSH (1)                                         Git Manual

 💡 DESCRIPTION
    ・自分のコミットをリモートリポジトリにアップロードする
    ・ローカルのブランチをリモートに公開する
    
    ※ GitGymではシミュレーションであり、実際のネットワーク送信は行われません。

 📋 SYNOPSIS
    git push [<remote>] [<branch>]

 ⚙️  COMMON OPTIONS
    -f, --force
        強制的にプッシュします（リモートの履歴を上書きする可能性があります）。
`
}
