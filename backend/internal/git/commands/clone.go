package commands

// clone.go - Simulated Git Clone Command
//
// IMPORTANT: This implementation does NOT clone from real network URLs.
// It looks up SharedRemotes (pre-ingested virtual remotes) or creates
// a simulated remote from the URL. Objects are copied in-memory.

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("clone", func() git.Command { return &CloneCommand{} })
}

type CloneCommand struct{}

// SafeRepoNameRegex enforces alphanumeric names to prevent traversal
var SafeRepoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

type CloneOptions struct {
	URL string
}

type cloneContext struct {
	RepoName   string
	RemoteRepo *gogit.Repository
	RemoteSt   storage.Storer
	RemotePath string
}

func (c *CloneCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {

	log.Printf("Clone: Starting execution args=%v", args)

	s.Lock()
	defer s.Unlock()

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		return "", err
	}

	// 2. Resolve Context (Repo Name, Remote Source)
	clCtx, err := c.resolveContext(s, opts)
	if err != nil {
		return "", err
	}

	// 3. Perform Clone
	return c.performClone(s, clCtx)
}

func (c *CloneCommand) parseArgs(args []string) (*CloneOptions, error) {
	opts := &CloneOptions{}
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if opts.URL == "" {
				opts.URL = arg
			}
		}
	}

	if opts.URL == "" {
		return nil, fmt.Errorf("usage: git clone <url>")
	}
	return opts, nil
}

func (c *CloneCommand) resolveContext(s *git.Session, opts *CloneOptions) (*cloneContext, error) {
	// Extract repo name from URL
	parts := strings.Split(opts.URL, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid url")
	}
	repoName := parts[len(parts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")

	// SECURITY: Input Validation
	if !SafeRepoNameRegex.MatchString(repoName) {
		return nil, fmt.Errorf("invalid repository name '%s': must contain only alphanumeric characters, underscores, or hyphens", repoName)
	}
	if repoName == "." || repoName == ".." {
		return nil, fmt.Errorf("invalid repository name: cannot be relative path")
	}

	if _, exists := s.Repos[repoName]; exists {
		return nil, fmt.Errorf("destination path '%s' already exists and is not an empty directory", repoName)
	}

	// Resolve Remote Repository
	var remoteRepo *gogit.Repository
	var remoteSt storage.Storer
	var remotePath string

	if s.Manager != nil {
		// Check SharedRemotes
		log.Printf("Clone: Checking shared remotes for %s", opts.URL)

		if r, ok := s.Manager.GetSharedRemote(opts.URL); ok {
			log.Printf("Clone: Found shared remote for URL %s", opts.URL)
			remoteRepo = r
			remoteSt = r.Storer

			s.Manager.RLock()
			path, found := s.Manager.SharedRemotePaths[opts.URL]
			s.Manager.RUnlock()
			if found {
				remotePath = path
			} else {
				remotePath = opts.URL
			}
		} else if r, ok := s.Manager.GetSharedRemote(repoName); ok {
			log.Printf("Clone: Found shared remote by name %s", repoName)
			remoteRepo = r
			remoteSt = r.Storer

			s.Manager.RLock()
			path, found := s.Manager.SharedRemotePaths[repoName]
			s.Manager.RUnlock()
			if found {
				remotePath = path
			} else {
				remotePath = repoName
			}
		}
	}

	if remoteRepo == nil {
		return nil, fmt.Errorf("repository '%s' not found in shared remotes. Network cloning is disabled to prevent timeout issues. Please use a valid shared remote URL.", opts.URL)
	}

	return &cloneContext{
		RepoName:   repoName,
		RemoteRepo: remoteRepo,
		RemoteSt:   remoteSt,
		RemotePath: remotePath,
	}, nil
}

func (c *CloneCommand) performClone(s *git.Session, clCtx *cloneContext) (string, error) {
	log.Printf("Clone: Remote resolved. Path: %s. Starting Local Creation...", clCtx.RemotePath)

	// Create Local Working Copy
	if errMkdir := s.Filesystem.MkdirAll(clCtx.RepoName, 0755); errMkdir != nil {
		return "", fmt.Errorf("failed to create directory: %w", errMkdir)
	}

	repoFS, err := s.Filesystem.Chroot(clCtx.RepoName)
	if err != nil {
		return "", fmt.Errorf("failed to chroot: %w", err)
	}

	// Create .git
	if errDotGit := repoFS.MkdirAll(".git", 0755); errDotGit != nil {
		return "", fmt.Errorf("failed to create .git directory: %w", errDotGit)
	}
	dotGitFS, err := repoFS.Chroot(".git")
	if err != nil {
		return "", fmt.Errorf("failed to chroot .git: %w", err)
	}

	localSt := filesystem.NewStorage(dotGitFS, cache.NewObjectLRUDefault())
	hybridSt := git.NewHybridStorer(localSt, clCtx.RemoteSt)

	localRepo, err := gogit.Init(hybridSt, repoFS)
	if err != nil {
		return "", fmt.Errorf("failed to init local repo: %w", err)
	}

	log.Printf("Clone: Using HybridStorer (Zero-Copy). Local initialized.")

	// Copy References
	if err := c.copyReferences(localRepo, clCtx.RemoteRepo); err != nil {
		log.Printf("Clone: Warning - Issue copying references: %v", err)
	}

	// Configure Origin
	// Use the raw path or URL? Logic logic used clCtx.RemotePath as "originURL" var before? No, it used 'url' arg mostly.

	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{clCtx.RemotePath}, // Using resolved path
	})
	if err != nil {
		return "", fmt.Errorf("failed to configure origin: %w", err)
	}

	s.Repos[clCtx.RepoName] = localRepo

	// Auto-cd
	s.CurrentDir = "/" + clCtx.RepoName

	// Checkout Default Branch
	if err := c.checkoutDefaultBranch(localRepo, clCtx.RemoteRepo); err != nil {
		log.Printf("Clone: Warning - Checkout default branch issue: %v", err)
	}

	log.Printf("Clone: Success. Cloned into %s", clCtx.RepoName)
	return fmt.Sprintf("Cloned into '%s'... (Using shared remote)", clCtx.RepoName), nil
}

func (c *CloneCommand) copyReferences(local *gogit.Repository, remote *gogit.Repository) error {
	refs, err := remote.References()
	if err != nil {
		return err
	}
	return refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		if name.IsBranch() {
			newRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", name.Short()))
			newRef := plumbing.NewHashReference(newRefName, ref.Hash())
			return local.Storer.SetReference(newRef)
		} else if name.IsRemote() || name.IsTag() {
			return local.Storer.SetReference(ref)
		}
		return nil
	})
}

func (c *CloneCommand) checkoutDefaultBranch(local *gogit.Repository, remote *gogit.Repository) error {
	w, err := local.Worktree()
	if err != nil {
		return err
	}

	headRef, err := remote.Head()
	targetBranch := plumbing.ReferenceName("refs/heads/main")
	if err == nil {
		if headRef.Type() == plumbing.SymbolicReference {
			targetBranch = headRef.Target()
		} else if headRef.Name().IsBranch() {
			targetBranch = headRef.Name()
		}
	}

	shortName := targetBranch.Short()
	remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/origin/%s", shortName))

	if ref, err := local.Reference(remoteRefName, true); err == nil {
		newBranchRef := plumbing.NewHashReference(targetBranch, ref.Hash())
		_ = local.Storer.SetReference(newBranchRef)
		return w.Checkout(&gogit.CheckoutOptions{
			Branch: targetBranch,
			Force:  true,
		})
	}
	return fmt.Errorf("could not resolve default branch '%s'", shortName)
}

func (c *CloneCommand) Help() string {
	return `📘 GIT-CLONE (1)                                        Git Manual

 💡 DESCRIPTION
    リモートリポジトリを複製して、手元にローカルリポジトリを作成します。
    GitGymでは事前定義されたリポジトリURLのみサポートしています。

 📋 SYNOPSIS
    git clone <url>
`
}
