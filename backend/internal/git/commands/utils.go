package commands

import (
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Shared utilities for commands

func isFastForward(repo *gogit.Repository, oldHash, newHash plumbing.Hash) (bool, error) {
	// Check if oldHash is ancestor of newHash
	cNew, err := repo.CommitObject(newHash)
	if err != nil {
		return false, err
	}
	cOld, err := repo.CommitObject(oldHash)
	if err != nil {
		return false, err
	}

	bases, err := cNew.MergeBase(cOld)
	if err != nil {
		return false, err
	}

	// If one of the merge bases IS the old commit, then old is strictly reachable from new.
	for _, b := range bases {
		if b.Hash == oldHash {
			return true, nil
		}
	}

	return false, nil
}
