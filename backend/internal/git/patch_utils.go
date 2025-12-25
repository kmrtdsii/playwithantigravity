package git

// patch_utils.go - Shared Utilities for Commit Patch Operations
//
// This file contains common helper functions used by commands that need to
// apply changes from one commit onto a worktree (e.g., rebase, cherry-pick).
// Extracted to reduce code duplication and improve maintainability.

import (
	"fmt"
	"os"
	"strings"

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
	rev = strings.TrimSpace(rev)
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

			forEachErr := cIter.ForEach(func(c *object.Commit) error {
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
			if forEachErr != nil && forEachErr.Error() != "stop iteration" {
				return nil, forEachErr
			}

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

// ErrConflict is returned when a merge cannot be resolved automatically.
var ErrConflict = fmt.Errorf("merge conflict")

// Merge3Way performs a 3-way merge of files between Base, Ours, and Theirs commits
// and applies the result to the Worktree.
//
// Strategy:
// - Base == Ours && Base != Theirs -> Update to Theirs (Fast-forward/Apply change)
// - Base != Ours && Base == Theirs -> Keep Ours (Already applied or irrelevant)
// - Base != Ours && Base != Theirs && Ours == Theirs -> Keep Ours (Both made same change)
// - Base != Ours && Base != Theirs && Ours != Theirs -> CONFLICT
//
// In case of conflict, it writes conflict markers to the file and returns ErrConflict.
func Merge3Way(w *gogit.Worktree, base, ours, theirs *object.Commit) error {
	// 1. Collect all file paths from all 3 trees
	paths := make(map[string]struct{})

	collectPaths := func(c *object.Commit) error {
		if c == nil {
			return nil
		}
		fIter, err := c.Files()
		if err != nil {
			return err
		}
		return fIter.ForEach(func(f *object.File) error {
			paths[f.Name] = struct{}{}
			return nil
		})
	}

	if err := collectPaths(base); err != nil {
		return err
	}
	if err := collectPaths(ours); err != nil {
		return err
	}
	if err := collectPaths(theirs); err != nil {
		return err
	}

	hasConflict := false

	// 2. Iterate all paths
	for path := range paths {
		// Helper to get hash (zero hash if missing)
		getHashAndContent := func(c *object.Commit) (plumbing.Hash, string, error) {
			if c == nil {
				return plumbing.ZeroHash, "", nil
			}
			f, err := c.File(path)
			if err != nil {
				// File not found in commit
				return plumbing.ZeroHash, "", nil
			}
			content, err := f.Contents()
			if err != nil {
				return plumbing.ZeroHash, "", err
			}
			return f.Hash, content, nil
		}

		baseH, _, err := getHashAndContent(base)
		if err != nil {
			return err
		}
		oursH, oursContent, err := getHashAndContent(ours)
		if err != nil {
			return err
		}
		theirsH, theirsContent, err := getHashAndContent(theirs)
		if err != nil {
			return err
		}

		// Analysis
		if oursH == theirsH {
			// No divergence between Ours and Theirs. Keep Ours.
			continue
		}

		if baseH == oursH {
			if baseH != theirsH {
				// Ours didn't change, Theirs changed (or deleted).
				// Action: Update to Theirs.
				if theirsH == plumbing.ZeroHash {
					// Theirs deleted it.
					if err := w.Filesystem.Remove(path); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("failed to remove %s: %w", path, err)
					}
					w.Remove(path) // Stage removal
				} else {
					// Theirs modified/added it.
					if err := writeFile(w, path, theirsContent); err != nil {
						return err
					}
					w.Add(path)
				}
			} else {
				// Base == Ours == Theirs. Nothing to do.
			}
		} else {
			// Ours changed (or deleted) from Base.
			if baseH == theirsH {
				// Theirs didn't change. Keep Ours. (No-op)
			} else {
				// Both changed from Base, and Ours != Theirs.
				// CONFLICT.
				hasConflict = true
				conflictContent := fmt.Sprintf("<<<<<<< HEAD\n%s=======\n%s>>>>>>> %s\n", oursContent, theirsContent, theirs.Hash.String()[:7])
				if err := writeFile(w, path, conflictContent); err != nil {
					return err
				}
				// Do NOT stage (git behavior for conflicts)
			}
		}
	}

	if hasConflict {
		return ErrConflict
	}
	return nil
}

func writeFile(w *gogit.Worktree, path, content string) error {
	f, err := w.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}
