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

// Ensure CloneCommand implements git.Command
var _ git.Command = (*CloneCommand)(nil)

// SafeRepoNameRegex enforces alphanumeric names to prevent traversal
var SafeRepoNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

type CloneOptions struct {
	URL       string
	Directory string
}

type cloneContext struct {
	RepoName   string
	RemoteRepo *gogit.Repository
	RemoteSt   storage.Storer
	RemotePath string
	RemoteURL  string // The original requested URL (for display/config)
}

func (c *CloneCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {

	s.Lock()
	defer s.Unlock()

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
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
			} else if opts.Directory == "" {
				opts.Directory = arg
			}
		}
	}

	if opts.URL == "" {
		return nil, fmt.Errorf("usage: git clone <url> [<directory>]")
	}
	return opts, nil
}

func (c *CloneCommand) resolveContext(s *git.Session, opts *CloneOptions) (*cloneContext, error) {
	var repoName string

	if opts.Directory != "" {
		repoName = opts.Directory
	} else {
		// Extract repo name from URL
		parts := strings.Split(opts.URL, "/")
		if len(parts) == 0 {
			return nil, fmt.Errorf("invalid url")
		}
		repoName = parts[len(parts)-1]
		repoName = strings.TrimSuffix(repoName, ".git")
	}

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
		if r, ok := s.Manager.GetSharedRemote(opts.URL); ok {
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
			// This fallback might be ambiguous if repoName is custom 'my-project' but remote is 'repo'
			// Only rely on URL matching if possible, but keep fallback for short names
			// However, if directory is custom, repoName is custom. Remote lookup should use URL mainly.
			// But keeping logic for now.
			remoteRepo = r
			remoteSt = r.Storer

			// ...
			remotePath = repoName
		}
	}

	if remoteRepo == nil {
		return nil, fmt.Errorf("repository '%s' not found in shared remotes. Network cloning is disabled to prevent timeout issues. Please use a valid shared remote URL", opts.URL)
	}

	return &cloneContext{
		RepoName:   repoName,
		RemoteRepo: remoteRepo,
		RemoteSt:   remoteSt,
		RemotePath: remotePath,
		RemoteURL:  opts.URL,
	}, nil
}

func (c *CloneCommand) performClone(s *git.Session, clCtx *cloneContext) (string, error) {
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

	// Perform Full Object Copy (No HybridStorer)
	if err := c.copyObjects(clCtx.RemoteSt, localSt); err != nil {
		return "", fmt.Errorf("failed to copy objects: %w", err)
	}

	localRepo, err := gogit.Init(localSt, repoFS)
	if err != nil {
		return "", fmt.Errorf("failed to init local repo: %w", err)
	}

	// Copy References
	if err := c.copyReferences(localRepo, clCtx.RemoteRepo); err != nil {
		log.Printf("Clone: Warning - Issue copying references: %v", err)
	}

	// Configure Origin
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{clCtx.RemotePath}, // Use internal path for functionality
	})
	if err != nil {
		return "", fmt.Errorf("failed to configure origin: %w", err)
	}

	// Store the friendly URL for display purposes (git remote -v)
	cfg, err := localRepo.Config()
	if err == nil {
		cfg.Raw.Section("remote").Subsection("origin").AddOption("displayurl", clCtx.RemoteURL)
		if err := localRepo.Storer.SetConfig(cfg); err != nil {
			log.Printf("Clone: Warning - failed to set display URL config: %v", err)
		}
	}

	s.Repos[clCtx.RepoName] = localRepo

	// Auto-cd
	s.CurrentDir = "/" + clCtx.RepoName

	// Checkout Default Branch
	if err := c.checkoutDefaultBranch(localRepo, clCtx.RemoteRepo); err != nil {
		log.Printf("Clone: Warning - Checkout default branch issue: %v", err)
	}

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

func (c *CloneCommand) copyObjects(src storage.Storer, dst storage.Storer) error {
	// iterate all objects
	iter, err := src.IterEncodedObjects(plumbing.AnyObject)
	if err != nil {
		return err
	}

	return iter.ForEach(func(obj plumbing.EncodedObject) error {
		_, err := dst.SetEncodedObject(obj)
		return err
	})
}

func (c *CloneCommand) Help() string {
	return `📘 GIT-CLONE (1)                                        Git Manual

 💡 DESCRIPTION
    ・リモートリポジトリを複製して、手元にローカルリポジトリを作成します。
    ・GitGymでは事前定義されたリポジトリURLのみサポートしています。

 📋 SYNOPSIS
    git clone <url> [<directory>]

 🛠  PRACTICAL EXAMPLES
    1. 基本: リポジトリをクローン
       $ git clone git@github.com:org/repo.git

    2. 実践: ディレクトリ名を指定してクローン (Recommended)
       「リポジトリ名とは別のフォルダ名で作業したい」場合に使います。
       $ git clone git@github.com:org/repo.git my-project

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-clone
`
}
