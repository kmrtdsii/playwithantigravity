package commands

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
	git.RegisterCommand("fetch", func() git.Command { return &FetchCommand{} })
}

type FetchCommand struct{}

func (c *FetchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse Flags
	isDryRun := false
	var positionalArgs []string
	for i, arg := range args {
		if i == 0 {
			continue // skip "fetch"
		}
		if arg == "-n" || arg == "--dry-run" {
			isDryRun = true
		} else if strings.HasPrefix(arg, "-") {
			// ignore other flags
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Syntax: git fetch [remote]
	remoteName := "origin"
	if len(positionalArgs) > 0 {
		remoteName = positionalArgs[0]
	}

	rem, err := repo.Remote(remoteName)
	if err != nil {
		return "", fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
	}

	cfg := rem.Config()
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
	}

	if !ok {
		return "", fmt.Errorf("remote repository '%s' not found (simulated path or URL required)", url)
	}

	// Scan remote refs (branches) and fetch them
	refs, err := srcRepo.References()
	if err != nil {
		return "", err
	}

	updated := 0
	results := []string{fmt.Sprintf("From %s", url)}

	err = refs.ForEach(func(r *plumbing.Reference) error {
		if r.Name().IsBranch() {
			branchName := r.Name().Short()
			localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))

			// Check if update needed
			currentLocal, err := repo.Reference(localRefName, true)
			if err == nil && currentLocal.Hash() == r.Hash() {
				return nil // up to date
			}

			if isDryRun {
				results = append(results, fmt.Sprintf(" * [dry-run] %s -> %s/%s", branchName, remoteName, branchName))
				return nil
			}

			// 1. Copy Objects
			err = fetchCopyCommitRecursive(srcRepo, repo, r.Hash())
			if err != nil {
				return err
			}

			// 2. Update Local Reference: refs/remotes/<remote>/<branch>
			newRef := plumbing.NewHashReference(localRefName, r.Hash())
			err = repo.Storer.SetReference(newRef)
			if err != nil {
				return err
			}

			results = append(results, fmt.Sprintf(" * [%s] %s -> %s/%s",
				func() string {
					if err != nil {
						return "new branch"
					}
					return "updated"
				}(),
				branchName, remoteName, branchName))
			updated++
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if updated == 0 && !isDryRun {
		return results[0] + "\nAlready up to date.", nil
	}

	return strings.Join(results, "\n"), nil
}

func (c *FetchCommand) Help() string {
	return "usage: git fetch [remote]"
}

// Duplicated helpers for Fetch (different direction)

func fetchCopyCommitRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if fetchHasObject(dst, hash) {
		return nil
	}

	obj, err := src.Storer.EncodedObject(plumbing.CommitObject, hash)
	if err != nil {
		return err
	}
	_, err = dst.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	commit, err := object.DecodeCommit(src.Storer, obj)
	if err != nil {
		return err
	}

	for _, p := range commit.ParentHashes {
		if err := fetchCopyCommitRecursive(src, dst, p); err != nil {
			return err
		}
	}
	return fetchCopyTreeRecursive(src, dst, commit.TreeHash)
}

func fetchCopyTreeRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if fetchHasObject(dst, hash) {
		return nil
	}
	obj, err := src.Storer.EncodedObject(plumbing.TreeObject, hash)
	if err != nil {
		return err
	}
	_, err = dst.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	tree, err := object.DecodeTree(src.Storer, obj)
	if err != nil {
		return err
	}

	for _, entry := range tree.Entries {
		if entry.Mode.IsFile() {
			if err := fetchCopyBlob(src, dst, entry.Hash); err != nil {
				return err
			}
		} else if entry.Mode == 0040000 { // Dir
			if err := fetchCopyTreeRecursive(src, dst, entry.Hash); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetchCopyBlob(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if fetchHasObject(dst, hash) {
		return nil
	}
	obj, err := src.Storer.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		return err
	}
	_, err = dst.Storer.SetEncodedObject(obj)
	return err
}

func fetchHasObject(repo *gogit.Repository, hash plumbing.Hash) bool {
	_, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	return err == nil
}
