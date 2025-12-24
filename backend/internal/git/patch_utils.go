package git

// patch_utils.go - Shared Utilities for Commit Patch Operations
//
// This file contains common helper functions used by commands that need to
// apply changes from one commit onto a worktree (e.g., rebase, cherry-pick).
// Extracted to reduce code duplication and improve maintainability.

import (
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ApplyCommitChanges applies the changes introduced by a commit onto the current worktree.
// It computes the diff between the commit and its parent, then applies those changes.
// For root commits (no parent), all files in the commit are added.
//
// Parameters:
//   - w: The worktree to apply changes to
//   - commit: The commit whose changes should be applied
//
// Returns an error if the patch computation or file operations fail.
func ApplyCommitChanges(w *gogit.Worktree, commit *object.Commit) error {
	var parentTree *object.Tree

	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return fmt.Errorf("failed to get parent commit: %w", err)
		}
		parentTree, err = parent.Tree()
		if err != nil {
			return fmt.Errorf("failed to get parent tree: %w", err)
		}
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("failed to get commit tree: %w", err)
	}

	// Handle root commit (no parent)
	if parentTree == nil {
		return applyRootCommitFiles(w, commit)
	}

	// Compute and apply patch
	patch, err := parentTree.Patch(commitTree)
	if err != nil {
		return fmt.Errorf("failed to compute patch: %w", err)
	}

	return applyPatchToWorktree(w, commit, patch)
}

// applyRootCommitFiles handles the special case of applying a root commit's files.
// All files in the commit are written to the worktree.
func applyRootCommitFiles(w *gogit.Worktree, commit *object.Commit) error {
	files, err := commit.Files()
	if err != nil {
		return fmt.Errorf("failed to get commit files: %w", err)
	}

	return files.ForEach(func(f *object.File) error {
		content, err := f.Contents()
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", f.Name, err)
		}

		wFile, err := w.Filesystem.OpenFile(f.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open file %s for writing: %w", f.Name, err)
		}
		defer wFile.Close()

		if _, err := wFile.Write([]byte(content)); err != nil {
			return fmt.Errorf("failed to write file %s: %w", f.Name, err)
		}

		if _, err := w.Add(f.Name); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", f.Name, err)
		}

		return nil
	})
}

// applyPatchToWorktree applies a computed patch to the worktree.
func applyPatchToWorktree(w *gogit.Worktree, commit *object.Commit, patch *object.Patch) error {
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()

		// File deletion
		if to == nil {
			if from != nil {
				if err := w.Filesystem.Remove(from.Path()); err != nil {
					// Ignore errors for files that don't exist
					if !os.IsNotExist(err) {
						return fmt.Errorf("failed to remove file %s: %w", from.Path(), err)
					}
				}
			}
			continue
		}

		// File addition or modification
		path := to.Path()
		file, err := commit.File(path)
		if err != nil {
			// File might not exist in this commit's tree (edge case)
			continue
		}

		content, err := file.Contents()
		if err != nil {
			continue
		}

		f, err := w.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open file %s for writing: %w", path, err)
		}

		if _, err := f.Write([]byte(content)); err != nil {
			f.Close()
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
		f.Close()

		if _, err := w.Add(path); err != nil {
			return fmt.Errorf("failed to stage file %s: %w", path, err)
		}
	}

	return nil
}

// ResolveRevision resolves a revision string (branch, tag, commit hash, short hash)
// to a full commit hash. Supports abbreviated commit hashes (>= 4 characters).
//
// Parameters:
//   - repo: The repository to resolve against
//   - rev: The revision string to resolve
//
// Returns the resolved hash or an error if not found.
func ResolveRevision(repo *gogit.Repository, rev string) (*plumbing.Hash, error) {
	// 1. Try standard resolution (branch, tag, full hash)
	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err == nil {
		return hash, nil
	}

	// 2. Try short hash resolution
	if len(rev) >= 4 && len(rev) < 40 {
		cIter, iterErr := repo.CommitObjects()
		if iterErr == nil {
			var match *plumbing.Hash
			found := false
			ambiguous := false

			cIter.ForEach(func(c *object.Commit) error {
				hashStr := c.Hash.String()
				if len(hashStr) >= len(rev) && hashStr[:len(rev)] == rev {
					if found {
						ambiguous = true
						return fmt.Errorf("stop iteration")
					}
					h := c.Hash // Copy to avoid pointer issues
					match = &h
					found = true
				}
				return nil
			})

			if ambiguous {
				return nil, fmt.Errorf("short commit hash '%s' is ambiguous", rev)
			}
			if found && match != nil {
				return match, nil
			}
		}
	}

	return nil, fmt.Errorf("revision '%s' not found", rev)
}
