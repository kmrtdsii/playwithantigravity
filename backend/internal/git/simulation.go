package git

import (
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ObjectUtils provides helpers for simulating git object transfer between repositories (in-memory).

// CopyCommitRecursive copies a commit and all its dependencies (parents, trees, blobs) from src to dst.
func CopyCommitRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	// Check if dst has it
	if HasObject(dst, hash) {
		return nil
	}

	// Get from Source
	obj, err := src.Storer.EncodedObject(plumbing.CommitObject, hash)
	if err != nil {
		return err
	}

	// Write to Dest
	_, err = dst.Storer.SetEncodedObject(obj)
	if err != nil {
		return err
	}

	// Decode to parse tree/parents
	commit, err := object.DecodeCommit(src.Storer, obj)
	if err != nil {
		return err
	}

	// Recurse parents
	for _, p := range commit.ParentHashes {
		if err := CopyCommitRecursive(src, dst, p); err != nil {
			return err
		}
	}

	// Copy Tree
	return CopyTreeRecursive(src, dst, commit.TreeHash)
}

// CopyTreeRecursive copies a tree and all its entries (blobs, subtrees) from src to dst.
func CopyTreeRecursive(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if HasObject(dst, hash) {
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
		if entry.Mode == 0160000 {
			// Submodule (commit), ignore or handle?
			continue
		}
		if entry.Mode.IsFile() {
			if err := CopyBlob(src, dst, entry.Hash); err != nil {
				return err
			}
		} else {
			if err := CopyTreeRecursive(src, dst, entry.Hash); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyBlob copies a blob object from src to dst.
func CopyBlob(src, dst *gogit.Repository, hash plumbing.Hash) error {
	if HasObject(dst, hash) {
		return nil
	}
	obj, err := src.Storer.EncodedObject(plumbing.BlobObject, hash)
	if err != nil {
		return err
	}
	_, err = dst.Storer.SetEncodedObject(obj)
	return err
}

// HasObject checks if a repository has a specific object.
func HasObject(repo *gogit.Repository, hash plumbing.Hash) bool {
	_, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	return err == nil
}

// IsFastForward checks if newHash is a fast-forward of oldHash.
func IsFastForward(repo *gogit.Repository, oldHash, newHash plumbing.Hash) (bool, error) {
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
