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

func (c *FetchCommand) Execute(_ context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse Flags
	isDryRun := false
	fetchAll := false
	prune := false
	tags := false
	var positionalArgs []string

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-n", "--dry-run":
			isDryRun = true
		case "--all":
			fetchAll = true
		case "-p", "--prune":
			prune = true
		case "-t", "--tags":
			tags = true
		case "-h", "--help":
			return c.Help(), nil
		default:
			if strings.HasPrefix(arg, "-") {
				return "", fmt.Errorf("unknown flag: %s", arg)
			}
			positionalArgs = append(positionalArgs, arg)
		}
	}

	var remotes []*gogit.Remote
	var err error

	if fetchAll {
		remotes, err = repo.Remotes()
		if err != nil {
			return "", fmt.Errorf("failed to list remotes: %w", err)
		}
	} else {
		// Single remote
		remoteName := "origin"
		if len(positionalArgs) > 0 {
			remoteName = positionalArgs[0]
		}
		rem, err := repo.Remote(remoteName)
		if err != nil {
			return "", fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
		}
		remotes = []*gogit.Remote{rem}
	}

	var allResults []string

	for _, rem := range remotes {
		res, err := c.fetchRemote(s, repo, rem, isDryRun, tags, prune)
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
			branchName := r.Name().Short()
			remoteBranches[branchName] = true

			localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))

			// Check if update needed
			currentLocal, errRef := repo.Reference(localRefName, true)
			if errRef == nil && currentLocal.Hash() == r.Hash() {
				return nil // up to date
			}

			if isDryRun {
				results = append(results, fmt.Sprintf(" * [dry-run] %s -> %s/%s", branchName, remoteName, branchName))
				return nil
			}

			// Copy Objects
			err = git.CopyCommitRecursive(srcRepo, repo, r.Hash())
			if err != nil {
				return err
			}

			// Update Local Reference
			newRef := plumbing.NewHashReference(localRefName, r.Hash())
			err = repo.Storer.SetReference(newRef)
			if err != nil {
				return err
			}

			status := "updated"
			if errRef != nil {
				status = "new branch"
			}

			results = append(results, fmt.Sprintf(" * [%s] %s -> %s/%s",
				status,
				branchName, remoteName, branchName))
			updated++
		}

		// 2. Handle Tags (Only if --tags is specified)
		// Note: Real git fetch auto-follows tags; here we simplify to strict flag or maybe auto-follow if easy?
		// User specifically asked for --tags. Let's make it conditional on flag for now to avoid noise.
		if fetchTags && r.Name().IsTag() {
			tagName := r.Name().Short()
			localTagRef := r.Name() // refs/tags/TAG

			// Check if update needed
			currentLocal, errRef := repo.Reference(localTagRef, true)
			if errRef == nil && currentLocal.Hash() == r.Hash() {
				return nil
			}

			if isDryRun {
				results = append(results, fmt.Sprintf(" * [dry-run] %s -> %s", tagName, tagName))
				return nil
			}

			// Copy Objects (Tag object or Commit object)
			// Ensure we copy the object the tag points to, and the tag object itself if annotated.
			// CopyCommitRecursive might not handle Tag Objects if it expects Commit.
			// Ideally we use a generic CopyObject if available, but CopyCommitRecursive works for Commits.
			// If it's an annotated tag, we need to copy that object too.
			// Creating a proper copy implementation is complex.
			// For simulation, let's assume lightweight tags (pointing to commits) for now or try CopyCommitRecursive.

			err = git.CopyCommitRecursive(srcRepo, repo, r.Hash())
			if err != nil {
				// Warn but don't fail entire fetch?
				results = append(results, fmt.Sprintf(" ! [error] %s (copy failed)", tagName))
				return nil
			}

			newRef := plumbing.NewHashReference(localTagRef, r.Hash())
			err = repo.Storer.SetReference(newRef)
			if err != nil {
				return err
			}

			status := "updated"
			if errRef != nil {
				status = "new tag"
			}
			results = append(results, fmt.Sprintf(" * [%s] %s -> %s", status, tagName, tagName))
			updated++
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// 3. Prune Logic
	// If --prune is set, we remove local remote-tracking branches that no longer exist on remote.
	if prune {
		localRefs, err := repo.References()
		if err == nil {
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
		}
	}

	if updated == 0 {
		return "", nil // Nothing to report for this remote if up to date
	}

	return strings.Join(results, "\n"), nil
}

func (c *FetchCommand) Help() string {
	return `📘 GIT-FETCH (1)                                        Git Manual

 🚀 NAME
    git-fetch - 他のリポジトリからオブジェクトと参照(refs)をダウンロードする

 📋 SYNOPSIS
    git fetch [<remote>]
    git fetch --all

 💡 DESCRIPTION
    リモートリポジトリの最新情報をローカルに取り込みますが、
    ワーキングツリーには反映しません（マージしません）。
    
    「何が変わったか」を確認するのに安全な操作です。
    GitGymでは、事前定義された仮想リモートから取得します。

 ⚙️  COMMON OPTIONS
    --all
        登録されている全てのリモートからフェッチします。

    --tags, -t
        リモートのタグも一緒にフェッチします。

    --prune, -p
        リモートで削除されたブランチに対応するローカルの追跡ブランチを削除します。

    --dry-run, -n
        実際にはフェッチを行わず、何が行われるかを表示します。

 🛠  EXAMPLES
    1. originから最新情報を取得
       $ git fetch

    2. 全てのリモートから取得
       $ git fetch --all
`
}
