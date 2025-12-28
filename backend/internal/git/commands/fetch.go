package commands

// fetch.go - Simulated Git Fetch Command
//
// IMPORTANT: This implementation does NOT perform actual network operations.
// It copies objects from in-memory virtual remotes (SharedRemotes or session-local).

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("fetch", func() git.Command { return &FetchCommand{} })
}

type FetchCommand struct{}

// Ensure FetchCommand implements git.Command
var _ git.Command = (*FetchCommand)(nil)

type FetchOptions struct {
	DryRun   bool
	FetchAll bool
	Prune    bool
	Tags     bool
	Remotes  []string
}

func (c *FetchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Parse Arguments
	opts, err := c.parseArgs(args)
	if err != nil {
		if err.Error() == "help requested" {
			return c.Help(), nil
		}
		return "", err
	}

	// 2. Resolve Targets (List of Remotes)
	remotes, err := c.resolveFetchTargets(repo, opts)
	if err != nil {
		return "", err
	}

	// 3. Execution (Loop and Fetch)
	return c.executeFetch(s, repo, remotes, opts)
}

func (c *FetchCommand) parseArgs(args []string) (*FetchOptions, error) {
	opts := &FetchOptions{}
	cmdArgs := args[1:]
	for i, arg := range cmdArgs {
		switch arg {
		case "-n", "--dry-run":
			opts.DryRun = true
		case "--all":
			opts.FetchAll = true
		case "-p", "--prune":
			opts.Prune = true
		case "-t", "--tags":
			opts.Tags = true
		case "-h", "--help":
			return nil, fmt.Errorf("help requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			// Only append positional args if not skipping (handled by index loop if manual, but range is safer here if we don't skip)
			// Wait, mixed flags/args logic in legacy was strict order? No, loop index i:
			// legacy: for i := 0; i < len(cmdArgs); i++ ...
			// Here range is fine unless we need to skip next arg (not needed for boolean flags).
			// If we had value flags, we'd need manual index handling. All current flags are boolean.
			opts.Remotes = append(opts.Remotes, arg)
		}
		_ = i
	}
	return opts, nil
}

func (c *FetchCommand) resolveFetchTargets(repo *gogit.Repository, opts *FetchOptions) ([]*gogit.Remote, error) {
	if opts.FetchAll {
		remotes, err := repo.Remotes()
		if err != nil {
			return nil, fmt.Errorf("failed to list remotes: %w", err)
		}
		return remotes, nil
	}

	// Single remote (default origin)
	remoteName := "origin"
	if len(opts.Remotes) > 0 {
		remoteName = opts.Remotes[0]
	}
	rem, err := repo.Remote(remoteName)
	if err != nil {
		return nil, fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
	}
	return []*gogit.Remote{rem}, nil
}

func (c *FetchCommand) executeFetch(s *git.Session, repo *gogit.Repository, remotes []*gogit.Remote, opts *FetchOptions) (string, error) {
	var allResults []string

	for _, rem := range remotes {
		res, err := c.fetchRemote(s, repo, rem, opts.DryRun, opts.Tags, opts.Prune)
		if err != nil {
			allResults = append(allResults, fmt.Sprintf("error: fetching %s: %v", rem.Config().Name, err))
		} else {
			if res != "" {
				allResults = append(allResults, res)
			}
		}
	}

	if len(allResults) == 0 {
		return "Already up to date.", nil
	}

	return strings.Join(allResults, "\n"), nil
}

func (c *FetchCommand) fetchRemote(s *git.Session, repo *gogit.Repository, rem *gogit.Remote, isDryRun bool, fetchTags bool, prune bool) (string, error) {
	cfg := rem.Config()
	remoteName := cfg.Name
	if len(cfg.URLs) == 0 {
		return "", fmt.Errorf("remote %s has no URL defined", remoteName)
	}
	url := cfg.URLs[0]

	// Look up simulated remote
	lookupKey := strings.TrimPrefix(url, "/")

	var srcRepo *gogit.Repository
	var ok bool

	// Check Session-local
	srcRepo, ok = s.Repos[lookupKey]
	if !ok && s.Manager != nil {
		// Check Shared
		srcRepo, ok = s.Manager.SharedRemotes[lookupKey]
		// Fallback: Check using full URL
		if !ok {
			srcRepo, ok = s.Manager.SharedRemotes[url]
		}
	}

	if !ok {
		return "", fmt.Errorf("remote repository '%s' not found (simulated path or URL required)", url)
	}

	// Scan remote refs (branches and tags)
	refs, err := srcRepo.References()
	if err != nil {
		return "", err
	}

	updated := 0
	results := []string{fmt.Sprintf("From %s", url)}

	// Track present remote branches for pruning later
	remoteBranches := make(map[string]bool)

	err = refs.ForEach(func(r *plumbing.Reference) error {
		// 1. Handle Branches
		if r.Name().IsBranch() {
			res, count, err := c.handleFetchBranch(repo, srcRepo, r, remoteName, isDryRun)
			if err != nil {
				return err
			}
			if res != "" {
				results = append(results, res)
			}
			updated += count
		}

		// 2. Handle Tags
		if fetchTags && r.Name().IsTag() {
			res, count, err := c.handleFetchTag(repo, srcRepo, r, isDryRun)
			if err != nil {
				// Warn but don't fail entire fetch?
				results = append(results, fmt.Sprintf(" ! [error] %s (copy failed)", r.Name().Short()))
				return nil
			}
			if res != "" {
				results = append(results, res)
			}
			updated += count
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	// 3. Prune Logic
	// If --prune is set, we remove local remote-tracking branches that no longer exist on remote.
	if prune {
		count, res, err := c.pruneRemoteBranches(repo, remoteName, remoteBranches, isDryRun)
		if err != nil {
			// Don't fail the whole fetch for prune errors
			// for now we ignore it
			_ = err
		}
		if len(res) > 0 {
			results = append(results, res...)
		}
		updated += count
	}

	if updated == 0 {
		return "", nil // Nothing to report for this remote if up to date
	}

	return strings.Join(results, "\n"), nil
}

func (c *FetchCommand) handleFetchBranch(repo, srcRepo *gogit.Repository, r *plumbing.Reference, remoteName string, isDryRun bool) (string, int, error) {
	branchName := r.Name().Short()
	localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))

	// Check if update needed
	currentLocal, errRef := repo.Reference(localRefName, true)
	if errRef == nil && currentLocal.Hash() == r.Hash() {
		return "", 0, nil // up to date
	}

	if isDryRun {
		return fmt.Sprintf(" * [dry-run] %s -> %s/%s", branchName, remoteName, branchName), 0, nil
	}

	// Copy Objects
	err := git.CopyCommitRecursive(srcRepo, repo, r.Hash())
	if err != nil {
		return "", 0, err
	}

	// Update Local Reference
	newRef := plumbing.NewHashReference(localRefName, r.Hash())
	err = repo.Storer.SetReference(newRef)
	if err != nil {
		return "", 0, err
	}

	status := "updated"
	if errRef != nil {
		status = "new branch"
	}

	return fmt.Sprintf(" * [%s] %s -> %s/%s", status, branchName, remoteName, branchName), 1, nil
}

func (c *FetchCommand) handleFetchTag(repo, srcRepo *gogit.Repository, r *plumbing.Reference, isDryRun bool) (string, int, error) {
	tagName := r.Name().Short()
	localTagRef := r.Name()

	// Check if update needed
	currentLocal, errRef := repo.Reference(localTagRef, true)
	if errRef == nil && currentLocal.Hash() == r.Hash() {
		return "", 0, nil
	}

	if isDryRun {
		return fmt.Sprintf(" * [dry-run] %s -> %s", tagName, tagName), 0, nil
	}

	// Copy Objects
	err := git.CopyCommitRecursive(srcRepo, repo, r.Hash())
	if err != nil {
		return "", 0, err
	}

	newRef := plumbing.NewHashReference(localTagRef, r.Hash())
	err = repo.Storer.SetReference(newRef)
	if err != nil {
		return "", 0, err
	}

	status := "updated"
	if errRef != nil {
		status = "new tag"
	}
	return fmt.Sprintf(" * [%s] %s -> %s", status, tagName, tagName), 1, nil
}

func (c *FetchCommand) Help() string {
	return `📘 GIT-FETCH (1)                                        Git Manual

 💡 DESCRIPTION
    ・リモートリポジトリから最新の情報をダウンロードする
    （ワーキングツリーのファイルは更新されません。あくまで情報取得のみです）
    
    「何が変わったか」を確認するのに安全な操作です。
    取得した情報は ` + "`" + `git log origin/main` + "`" + ` などで確認できます。

 📋 SYNOPSIS
    git fetch [<remote>] [<branch>]
    git fetch --all
    git fetch --prune

 ⚙️  COMMON OPTIONS
    --all
        登録されている全てのリモートからフェッチします。

    --tags, -t
        リモートのタグも一緒にフェッチします。

    --prune, -p
        リモートで削除されたブランチに対応するローカルの追跡ブランチを削除します。
        （これをやらないと、ローカルに古い origin/xxx が残り続けます）

    --dry-run, -n
        実際にはフェッチを行わず、何が行われるかを表示します。

 🛠  PRACTICAL EXAMPLES
    1. 基本: originから最新情報を取得
       $ git fetch

    2. 実践: 情報を整理しながら取得 (Recommended)
       「リモートで消されたブランチは、ローカルの追跡情報からも消す」
       $ git fetch -p

    3. 実践: 特定のブランチだけ取得
       「mainの更新だけ欲しい」という時に。
       $ git fetch origin main
`
}
func (c *FetchCommand) pruneRemoteBranches(repo *gogit.Repository, remoteName string, remoteBranches map[string]bool, isDryRun bool) (int, []string, error) {
	var results []string
	updated := 0
	localRefs, err := repo.References()
	if err != nil {
		return 0, nil, err
	}

	prefix := fmt.Sprintf("refs/remotes/%s/", remoteName)
	_ = localRefs.ForEach(func(r *plumbing.Reference) error {
		name := r.Name().String()
		if strings.HasPrefix(name, prefix) {
			// e.g. refs/remotes/origin/main -> branchName = main
			branchName := strings.TrimPrefix(name, prefix)
			if !remoteBranches[branchName] {
				// Stale!
				if isDryRun {
					results = append(results, fmt.Sprintf(" - [dry-run] [deleted] (none) -> %s/%s", remoteName, branchName))
				} else {
					err := repo.Storer.RemoveReference(r.Name())
					if err != nil {
						results = append(results, fmt.Sprintf(" ! [error] %s/%s (prune failed)", remoteName, branchName))
					} else {
						results = append(results, fmt.Sprintf(" - [deleted] (none) -> %s/%s", remoteName, branchName))
						updated++
					}
				}
			}
		}
		return nil
	})
	return updated, results, nil
}
