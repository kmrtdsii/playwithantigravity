package commands

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kmrtdsii/playwithantigravity/backend/internal/git"
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

	// Syntax: git fetch [remote]
	remoteName := "origin"
	if len(args) > 1 {
		remoteName = args[1]
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
	srcRepo, ok := s.Repos[url]
	if !ok {
		return "", fmt.Errorf("remote repository '%s' not found in simulation session", url)
	}

	// Scan remote refs (branches) and fetch them
	// We map remote branch refs/heads/X to local refs/remotes/<remote>/X

	refs, err := srcRepo.References()
	if err != nil {
		return "", err
	}

	updated := 0

	err = refs.ForEach(func(r *plumbing.Reference) error {
		if r.Name().IsBranch() {
			branchName := r.Name().Short()
			// Fetch Logic

			// 1. Copy Objects (Src -> Dst)
			// Using shared helpers (need to export or dup)
			// For now, duplicate copyCommitRecursive logic or move to shared util?
			// Duplicating for speed, minimal diff.
			err := fetchCopyCommitRecursive(srcRepo, repo, r.Hash())
			if err != nil {
				return err
			}

			// 2. Update Local Reference: refs/remotes/<remote>/<branch>
			localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))

			// Update logic (force update for fetch)
			newRef := plumbing.NewHashReference(localRefName, r.Hash())
			err = repo.Storer.SetReference(newRef)
			if err != nil {
				return err
			}
			updated++
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("From %s\n * [new branch/tag] -> %s/*", url, remoteName), nil
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
