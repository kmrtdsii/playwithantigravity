package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("restore", func() git.Command { return &RestoreCommand{} })
}

type RestoreCommand struct{}

func (c *RestoreCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	staged := false
	var files []string

	// Basic parsing
	for _, arg := range args {
		if arg == "restore" {
			continue
		}
		if arg == "--staged" {
			staged = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue // ignore other flags
		}
		files = append(files, arg)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("fatal: you must specify path(s) to restore")
	}

	if staged {
		// restore --staged: Unstage files (reset index to HEAD)
		headRef, err := repo.Head()
		if err != nil {
			// No HEAD (initial commit?), unstaging means removing from index
			// We can iterate files and remove them from index
			idx, idxErr := repo.Storer.Index()
			if idxErr != nil {
				return "", idxErr
			}
			for _, file := range files {
				// Remove file from index entries
				newEntries := make([]*index.Entry, 0, len(idx.Entries))
				for _, e := range idx.Entries {
					if e.Name != file {
						newEntries = append(newEntries, e)
					}
				}
				idx.Entries = newEntries
			}
			_ = repo.Storer.SetIndex(idx)
			return "Unstaged files (initial commit)", nil
		}

		// HEAD exists, copy HEAD entry to Index
		commit, err := repo.CommitObject(headRef.Hash())
		if err != nil {
			return "", err
		}

		tree, err := commit.Tree()
		if err != nil {
			return "", err
		}

		idx, err := repo.Storer.Index()
		if err != nil {
			return "", err
		}

		for _, file := range files {
			// 1. Check if file exists in HEAD
			entry, err := tree.File(file)
			if err != nil {
				// File not in HEAD (it was a new file added). Remove from Index.
				newEntries := make([]*index.Entry, 0, len(idx.Entries))
				for _, e := range idx.Entries {
					if e.Name != file {
						newEntries = append(newEntries, e)
					}
				}
				idx.Entries = newEntries
				continue
			}

			// 2. File exists in HEAD. Update Index to match HEAD.
			found := false
			for i, e := range idx.Entries {
				if e.Name == file {
					// Update
					e.Hash = entry.Hash
					e.Mode = entry.Mode
					// ModifiedAt, Size etc?
					idx.Entries[i] = e
					found = true
					break
				}
			}
			if !found {
				// If not in index but in HEAD, add it back
				idx.Entries = append(idx.Entries, &index.Entry{
					Name: file,
					Hash: entry.Hash,
					Mode: entry.Mode,
				})
			}
		}
		_ = repo.Storer.SetIndex(idx)
		return "Unstaged files", nil

	} else {
		// restore (worktree): Discard changes in worktree (restore from Index)

		// We need the Worktree to access the correct Filesystem
		w, err := repo.Worktree()
		if err != nil {
			return "", err
		}

		idx, err := repo.Storer.Index()
		if err != nil {
			return "", err
		}

		// Expand "." to all specific files in the index
		var targets []string

		// Determine current directory prefix relative to repo root
		prefix := ""

		containDot := false
		for _, f := range files {
			if f == "." {
				containDot = true
				break
			}
		}

		if containDot {
			// Add all index entries matching prefix
			for _, e := range idx.Entries {
				if strings.HasPrefix(e.Name, prefix) {
					targets = append(targets, e.Name)
				}
			}
		} else {
			targets = files
		}

		if len(targets) == 0 {
			if containDot {
				return "Nothing to restore (no tracked files in current directory)", nil
			}
		}

		restoredCount := 0
		for _, file := range targets {
			// Find entry in index
			var entry *index.Entry
			for _, e := range idx.Entries {
				if e.Name == file {
					entry = e
					break
				}
			}

			if entry == nil {
				if !containDot {
					return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", file)
				}
				continue
			}

			// Read blob from Object Storage
			blob, err := repo.BlobObject(entry.Hash)
			if err != nil {
				return "", fmt.Errorf("failed to read blob %s: %w", entry.Hash, err)
			}
			reader, err := blob.Reader()
			if err != nil {
				return "", err
			}
			defer reader.Close()

			// Write to Worktree using w.Filesystem (which is rooted at repo root)
			// instead of s.Filesystem (which is rooted at global root)
			f, err := w.Filesystem.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(f, reader); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
			restoredCount++
		}

		if containDot {
			return fmt.Sprintf("Restored all tracked files in current directory (%d files)", restoredCount), nil
		}
		return "Restored files in worktree", nil
	}
}

func (c *RestoreCommand) Help() string {
	return `📘 GIT-RESTORE (1)                                      Git Manual

 💡 DESCRIPTION
    ・ファイルの変更を破棄して、元の状態に戻す
    ・ステージングした変更を取り消す（--staged）
    
    「編集をやり直したい」時や「addを取り消したい」時に使います。

 📋 SYNOPSIS
    git restore [<options>] <pathspec>...

 ⚙️  COMMON OPTIONS
    --staged
        ワーキングツリーではなく、インデックス（ステージングエリア）を復元します。
        ` + "`git add`" + ` した内容を取り消す際によく使用します。

 🛠  EXAMPLES
    1. ワーキングツリーの変更を破棄する（元に戻す）
       $ git restore README.md

    2. ステージングした変更を取り消す（Unstage）
       $ git restore --staged README.md

 🔗 REFERENCE
    Full documentation: https://git-scm.com/docs/git-restore

 💡 TIPS
    ` + "`" + `git restore .` + "`" + ` を実行すると、現在のディレクトリ以下の
    「まだaddしていない変更」をすべて破棄します（Untrackedなファイルは消えません）。
    「実験的にいろいろいじったけど、全部なかったことにしてスッキリしたい」時に便利です。
`
}
